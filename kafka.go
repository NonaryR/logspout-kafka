package kafka

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/fsouza/go-dockerclient"
	"github.com/gliderlabs/logspout/router"
	"gopkg.in/Shopify/sarama.v1"
	"crypto/tls"
	"io/ioutil"
	"crypto/x509"
)

var host string

func init() {
	router.AdapterFactories.Register(NewKafkaAdapter, "kafka")
}

type KafkaAdapter struct {
	route    *router.Route
	brokers  []string
	topic    string
	producer sarama.AsyncProducer
	tmpl     *template.Template
	envs     map[string]string
	host     string
}

type KafkaMessage struct {
	*router.Message
	Envs  *map[string]string
	Info  *DockerInfo
	Level int32
}

type DockerInfo struct {
	ContainerId     string
	ContainerFullId string
	ContainerName   string
	ImageId         string
	ImageName       string
	HostId          string
	HostName        string
	HostAddr        string
	HostIp          string
	Command         string
	Stack           string // or project for docker compose
	Service         string // or service for docker compose
}

func NewKafkaAdapter(route *router.Route) (router.LogAdapter, error) {
	host = getHostFromDockerApi()
	brokers := readBrokers(route.Address)
	if len(brokers) == 0 {
		return nil, errorf("The Kafka broker host:port is missing. Did you specify it as a route address?")
	}

	otherBrokers := readBrokersFromOption(route.Options)
	for _, b := range otherBrokers {
		brokers = append(brokers, b)
	}

	topic := readTopic(route.Address, route.Options)
	if topic == "" {
		return nil, errorf("The Kafka topic is missing. Did you specify it as a route option?")
	}

	var err error
	var tmpl *template.Template
	funcMap := template.FuncMap{
		"json":      jsonMarshal,
		"timestamp": timestamp,
		"getenv":    tplGetEnvVar,
	}

	if kafkaTempl := os.Getenv("KAFKA_TEMPLATE"); kafkaTempl != "" {
		tmpl, err = template.New("kafka").Funcs(funcMap).Parse(kafkaTempl)
		if err != nil {
			return nil, errorf("Couldn't parse Kafka message template. %v", err)
		}
	}

	if os.Getenv("DEBUG") != "" {
		log.Printf("Starting Kafka producer for address: %s, topic: %s.\n", brokers, topic)
	}

	var retries int
	retries, err = strconv.Atoi(os.Getenv("KAFKA_CONNECT_RETRIES"))
	if err != nil {
		retries = 3
	}
	var producer sarama.AsyncProducer
	for i := 0; i < retries; i++ {
		producer, err = sarama.NewAsyncProducer(brokers, newConfig(route.Options))
		if err != nil {
			if os.Getenv("DEBUG") != "" {
				log.Println("Couldn't create Kafka producer. Retrying...", err)
			}
			if i == retries-1 {
				return nil, errorf("Couldn't create Kafka producer. %v", err)
			}
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	return &KafkaAdapter{
		route:    route,
		brokers:  brokers,
		topic:    topic,
		producer: producer,
		tmpl:     tmpl,
		envs:     getEnvMap(),
	}, nil
}
func getHostFromDockerApi() string {
	var client *docker.Client
	var err error
	if client, err = docker.NewClientFromEnv(); err != nil {
		return ""
	}
	var clientInfo *docker.DockerInfo
	if clientInfo, err = client.Info(); err != nil {
		return ""
	}
	return clientInfo.Name
}

func (a *KafkaAdapter) Stream(logstream chan *router.Message) {
	defer a.producer.Close()
	dockerInfo := &DockerInfo{}
	for rm := range logstream {
		message, err := a.formatMessage(rm, dockerInfo)
		if err != nil {
			log.Println("kafka:", err)
			a.route.Close()
			break
		}

		a.producer.Input() <- message
	}
}

func newConfig(options map[string]string) *sarama.Config {
	config := sarama.NewConfig()
	config.ClientID = "logspout"
	config.Producer.Return.Errors = false
	config.Producer.Return.Successes = false
	config.Producer.Flush.Frequency = 1 * time.Second
	config.Producer.RequiredAcks = sarama.WaitForLocal

	enableSsl, ssl := options["ssl"]
	if ssl && "true" == enableSsl {
		caPath := "/data/certs/kafka/ca.pem"
		certPath := "/data/certs/kafka/client_cert.pem"
		keyPath := "/data/certs/kafka/client_key.pem"

		log.Printf("Load certs: %s\n", caPath)
		ca, err := ioutil.ReadFile(caPath)
		if err != nil {
			log.Fatal("Could not load CA certificate!")
		}

		log.Printf("Load certs: %s\n", certPath)
		cert, err := ioutil.ReadFile(certPath)
		if err != nil {
			log.Fatal("Could not load clientCert!")
		}

		log.Printf("Load certs: %s\n", keyPath)
		key, err := ioutil.ReadFile(keyPath)
		if err != nil {
			log.Fatal("Could not load clientKey!")
		}
		keyPair, err := tls.X509KeyPair(cert, key)
		if err != nil {
			log.Fatal("Could not parser client cert! ", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(ca)

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{
			RootCAs:            caCertPool,
			Certificates:       []tls.Certificate{keyPair},
			InsecureSkipVerify: true,
		}
	}

	if opt := os.Getenv("KAFKA_COMPRESSION_CODEC"); opt != "" {
		switch opt {
		case "gzip":
			config.Producer.Compression = sarama.CompressionGZIP
		case "snappy":
			config.Producer.Compression = sarama.CompressionSnappy
		}
	}

	return config
}

func (a *KafkaAdapter) formatMessage(message *router.Message, dockerInfo *DockerInfo) (*sarama.ProducerMessage, error) {
	var encoder sarama.Encoder
	if a.tmpl != nil {
		prepareDockerInfo(message, dockerInfo)
		var w bytes.Buffer
		kafkaMessage := KafkaMessage{
			Message: message,
			Envs:    &a.envs,
			Info:    dockerInfo,
			Level:   getMessageLevel(message.Source),
		}
		if err := a.tmpl.Execute(&w, kafkaMessage); err != nil {
			return nil, err
		}
		encoder = sarama.ByteEncoder(w.Bytes())
	} else {
		encoder = sarama.StringEncoder(message.Data)
	}

	return &sarama.ProducerMessage{
		Topic: a.topic,
		Value: encoder,
	}, nil
}
func getMessageLevel(source string) int32 {
	if source == "stdout" {
		return 6 // LOG_INFO
	} else if source == "stderr" {
		return 3 // LOG_ERR
	} else {
		return 5 // LOG_NOTICE
	}
}

func prepareDockerInfo(message *router.Message, dockerInfo *DockerInfo) {
	if dockerInfo.ContainerFullId == message.Container.ID {
		return
	}
	dockerInfo.ContainerId = normalID(message.Container.ID)
	dockerInfo.ContainerFullId = message.Container.ID
	dockerInfo.ContainerName = normalName(message.Container.Name)
	dockerInfo.ImageId = message.Container.Image
	dockerInfo.ImageName = message.Container.Config.Image
	if label, ok := message.Container.Config.Labels["com.docker.stack.namespace"]; ok {
		dockerInfo.Stack = label
	} else {
		dockerInfo.Stack = message.Container.Config.Labels["com.docker.compose.project"]
	}
	if label, ok := message.Container.Config.Labels["com.docker.swarm.service.name"]; ok {
		dockerInfo.Service = label
	} else {
		dockerInfo.Service = message.Container.Config.Labels["com.docker.compose.service"]
	}
	if message.Container.Node != nil {
		dockerInfo.HostName = message.Container.Node.Name
		dockerInfo.HostAddr = message.Container.Node.Addr
		dockerInfo.HostIp = message.Container.Node.IP
	} else {
		dockerInfo.HostName = host
	}
	dockerInfo.Command = getCommand(message)
}

func getCommand(message *router.Message) string {
	terms := message.Container.Config.Entrypoint
	terms = append(terms, message.Container.Config.Cmd...)
	command := strings.Join(terms, " ")
	return command
}

func normalName(name string) string {
	return strings.TrimPrefix(name, "/")
}

func normalID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func tplGetEnvVar(env []string, key string) string {
	key_equals := key + "="
	for _, value := range env {
		if strings.HasPrefix(value, key_equals) {
			return value[len(key_equals):]
		}
	}
	return ""
}

func readBrokers(address string) []string {
	if strings.Contains(address, "/") {
		slash := strings.Index(address, "/")
		address = address[:slash]
	}

	return strings.Split(address, ",")
}

func readBrokersFromOption(options map[string]string) []string {
	brokers := options["brokers"]
	if brokers != "" {
		return strings.Split(brokers, "|")
	}
	return nil
}

func readTopic(address string, options map[string]string) string {
	var topic string
	if !strings.Contains(address, "/") {
		topic = options["topic"]
	} else {
		slash := strings.Index(address, "/")
		topic = address[slash+1:]
	}

	return topic
}

func errorf(format string, a ...interface{}) (err error) {
	err = fmt.Errorf(format, a...)
	if os.Getenv("DEBUG") != "" {
		fmt.Println(err.Error())
	}
	return
}

func jsonMarshal(v interface{}) (string) {
	jsonValue, _ := json.Marshal(v)
	return string(jsonValue)
}

func timestamp(t time.Time) (string) {
	return t.UTC().Format(time.RFC3339Nano)
}

func getEnvMap() map[string]string {
	var envs = make(map[string]string)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		envs[pair[0]] = pair[1]
	}
	return envs
}
