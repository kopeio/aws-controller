FROM debian:jessie

RUN apt-get update && apt-get install --yes ca-certificates

COPY /.build/artifacts/aws-controller /usr/bin/aws-controller

CMD /usr/bin/aws-controller

