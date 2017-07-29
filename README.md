# logspout-kafka

A [Logspout](https://github.com/gliderlabs/logspout) adapter for writing Docker container logs to [Kafka](https://github.com/apache/kafka) topics.

This project has extension for github fork upstream. More information refer [here](docker/README.md).

## usage

With *container-logs* as the Kafka topic for Docker container logs, we can direct all messages to Kafka using the **logspout** [Route API](https://github.com/gliderlabs/logspout/tree/master/routesapi):

```
curl http://container-host:8000/routes -d '{
  "adapter": "kafka",
  "filter_sources": ["stdout" ,"stderr"],
  "address": "kafka-broker1:9092,kafka-broker2:9092/container-logs"
}'
```

## message template configuration

The default behavior is to route the raw message to Kafka.  This adapter provides a mechanism for customizing the template to include additional metadata via the `KAFKA_TEMPLATE` environment variable.  Golang's [text/template](http://golang.org/pkg/text/template/) package is used for templating, where the model available for templating is the [Message struct](https://github.com/gliderlabs/logspout/blob/master/router/types.go).

The following example `KAFKA_TEMPLATE` configuration appends additional metadata to each log message received:
```
KAFKA_TEMPLATE="time=\"{{.Time}}\" container_name=\"{{.Container.Name}}\" source=\"{{.Source}}\" data=\"{{.Data}}\""
```

Example input and output
```
Hello World

time="2015-06-23 09:54:55.241951004 +0000 UTC" container_name="/hello_container" source="stdout" data="Hello World"
```

## route configuration

If you've mounted a volume to `/mnt/routes`, then consider pre-populating your routes. The following script configures a route to send standard messages from a "cat" container to one Kafka topic, and a route to send standard/error messages from a "dog" container to another topic.

```
cat > /logspout/routes/cat.json <<CAT
{
  "id": "cat",
  "adapter": "kafka",
  "filter_name": "cat_*",
  "filter_sources": ["stdout"],
  "address": "kafka-broker1:9092,kafka-broker2:9092/cat-logs"
}
CAT

cat > /logspout/routes/dog.json <<DOG
{
  "id": "dog",
  "adapter": "kafka",
  "filter_name": "dog_*",
  "filter_sources": ["stdout", "stderr"],
  "address": "kafka-broker1:9092,kafka-broker2:9092/dog-logs"
}
DOG

docker run --name logspout \
  -p "8000:8000" \
  --volume /logspout/routes:/mnt/routes \
  --volume /var/run/docker.sock:/tmp/docker.sock \
  fengzhou/logspout

```

The routes can be updated on a running container by using the **logspout** [Route API](https://github.com/gliderlabs/logspout/tree/master/routesapi) and specifying the route `id` "cat" or "dog".
