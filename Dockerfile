FROM alpine:latest

# NOTE: because of these args, if you want to build this manually you've to add
#       e.g. --build-arg TARGETARCH=amd64 to $ docker build ...

# "amd64" | "arm" | ...
ARG TARGETARCH
# usually empty. for "linux/arm/v7" => "v7"
ARG TARGETVARIANT

WORKDIR /varasto

# stores Varasto state (files' metadata)
VOLUME /varasto-db

ENTRYPOINT ["sto"]

CMD ["server"]

# symlink /root/varastoclient-config.json to /varasto-db/.. because it's stateful.
# this config is used for server subsystems (thumbnailing, FUSE projector) to communicate
# with the server.

RUN mkdir -p /varasto \
	&& ln -s /varasto/sto /bin/sto \
	&& ln -s /varasto-db/varastoclient-config.json /root/varastoclient-config.json \
	&& apk add --update smartmontools fuse \
	&& echo '{"db_location": "/varasto-db/varasto.db"}' > /varasto/config.json

COPY "rel/sto_linux-$TARGETARCH" /varasto/sto

ADD rel/public.tar.gz /varasto/

RUN chmod +x /varasto/sto
