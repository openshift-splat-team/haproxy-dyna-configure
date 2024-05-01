FROM registry.access.redhat.com/ubi8/go-toolset:1.20.10-10

USER root

WORKDIR /usr/src/app

COPY . .
RUN ls -lt
RUN go mod tidy && go mod vendor

COPY run.sh .

RUN ./hack/build.sh
CMD bin/haproxy-dyna-configure
