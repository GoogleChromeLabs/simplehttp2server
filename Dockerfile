FROM golang:1.8-onbuild

RUN mkdir /data
VOLUME ["/data"]
WORKDIR /data
EXPOSE 5000
ENTRYPOINT ["app"]
