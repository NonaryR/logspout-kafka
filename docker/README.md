# Change from github fork upstream
This docker contains logspout for kafka, syslog, httpstream, logstash.

For kafka, the special Info fields are exposed to simplify get all docker information, including host ip.

## KAFAK_TEMPLATE
Following example shows kafka template and using graylog format:
```
# Content in logspout.env via docker run --env-file option
KAFKA_TEMPLATE={"version": "1.1", "host": "{{.Info.HostName}}", "short_message": {{.Data | json}}, "timestamp": "{{.Time | timestamp }}", "level": {{.Level}}, "_container_id": "{{.Info.ContainerId}}", "_container_name": "{{.Info.ContainerName}}", "_message_source": "{{.Source}}", "_image_id": "{{.Info.ImageId}}", "_image_name": "{{.Info.ImageName}}", "_created": "{{.Container.Created | timestamp }}", "_host_id": "{{.Info.HostId}}", "_host_addr": "{{.Info.HostAddr}}", "_host_ip": "{{.Info.HostIp}}", "_command": {{.Info.Command | json }}, "_stack": "{{.Info.Stack}}", "_service": "{{.Info.Service}}" }
``` 

To support graylog `level` field, the field `.Level` is used.
- Standard Output: LOG_INFO (6)
- Standard Error: LOG_ERR (3)
- Others: LOG_NOTICE (5)

## Notes for docker swarm service
Since go template is also used for docker swarm service to evaluate environments, use escape if function (like json, timestamp) is used.
```
# Content in logspout.env via docker service --env-file option
KAFKA_TEMPLATE={{`{"version": "1.1", "host": "{{.Info.HostName}}", "_message_source": "{{.Source}}", "short_message": {{.Data | json}}, "timestamp": "{{.Time | timestamp }}", "level": {{.Level}}, "_container_id": "{{.Info.ContainerId}}", "_container_name": "{{.Info.ContainerName}}", "_container_hostname": "{{.Container.Config.Hostname}}", "_container_created": "{{.Container.Created | timestamp }}", "_container_command": {{.Info.Command | json }}, "_container_image_id": "{{.Info.ImageId}}", "_container_image_name": "{{.Info.ImageName}}", "_container_stack": "{{.Info.Stack}}", "_container_service": "{{.Info.Service}}" }`}} 
```

# Build

```shell
docker build -t <tag> .
```

# Usage (Based on github fork upstream)

```shell
docker run --rm --name="logspout" \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Info.ContainerName}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    fengzhou/logspout \
    kafka://kafkahost:9092?topic=container-logs

# multi kafka
docker run --rm --name="logspout" \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Info.ContainerName}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    fengzhou/logspout \
    kafka://broker1:9092?topic=container-logs&brokers=broker2:9092|broker3:9092

# enable ssl
# you need three certs in local directory and add ssl=true parameter
# eg.:
# /tmp/certs/kafka/
# ├── ca.pem
# ├── client_cert.pem
# └── client_key.pem
docker run --rm --name="logspout" \
    --volume=/var/run/docker.sock:/var/run/docker.sock \
    --volume=/tmp/certs/kafka:/data/certs/kafka \
    -e KAFKA_TEMPLATE="{\"time\":\"{{.Time | timestamp }}\",\"container_name\":\"{{.Info.ContainerName}}\",\"source\":\"{{.Source}}\",\"data\":{{.Data | json }}, \"origin\": \"{{.Data}}\" }" \
    fengzhou/logspout \
    kafka://broker1:9092?topic=container-logs&brokers=broker2:9092|broker3:9092&ssl=true
```