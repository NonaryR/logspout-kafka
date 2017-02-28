FROM gliderlabs/logspout:master

COPY ./build-go.sh /src/build-go.sh
RUN cd /src && ./build-go.sh "$(cat VERSION)-logstash-kafka"
