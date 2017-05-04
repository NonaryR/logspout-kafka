
# Build

```shell
docker build -t <tag> .
```

# Usage

```shell
docker run --rm --name="logspout" \
    --net=host \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Container.Name}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    caiqinzhou/logspout-kafka \
    kafka://localhost:9092?topic=container-logs

# multi kafka
docker run --rm --name="logspout" \
    --net=host \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Container.Name}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    caiqinzhou/logspout-kafka \
    kafka://broker1:9092?topic=container-logs&brokers=broker2:9092|broker3:9092

# enable ssl
# you need three certs in local directory and add ssl=true parameter
# eg.:
# /tmp/certs/kafka/
# ├── ca.pem
# ├── client_cert.pem
# └── client_key.pem
docker run --rm --name="logspout" \
    --net=host \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    --volume=/tmp/certs/kafka:/data/certs/kafka \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Container.Name}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    caiqinzhou/logspout-kafka \
    kafka://broker1:9092?topic=container-logs&brokers=broker2:9092|broker3:9092&ssl=true
```