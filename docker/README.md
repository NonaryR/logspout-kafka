
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
```