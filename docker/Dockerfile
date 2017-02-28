FROM gliderlabs/logspout:master

COPY ./build-go.sh /src/build-go.sh
RUN echo "$(date)"
RUN cd /src && ./build-go.sh "$(cat VERSION)-logstash-kafka"