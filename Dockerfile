# alpine:edge required for smartmontools 7.0
# alpine:latest (mine could be older) had 6.6
FROM alpine:edge

WORKDIR /varasto

VOLUME /varasto-db

ENTRYPOINT ["sto"]

CMD ["server"]

RUN mkdir -p /varasto \
	&& ln -s /varasto/sto /usr/local/bin/sto \
	&& apk add --update smartmontools fuse \
	&& echo '{"db_location": "/varasto-db/varasto.db"}' > /varasto/config.json \
	&& mkdir /tmp/fuse

COPY rel/sto_linux-amd64 /varasto/sto

ADD rel/public.tar.gz /varasto/

RUN chmod +x /varasto/sto
