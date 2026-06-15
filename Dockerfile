FROM ubuntu:24.04
WORKDIR /app

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/* \
    && mkdir -p /app/plugins

COPY BedrockPluginLoader /app/
COPY plugins/example.so /app/plugins/

RUN chmod +x /app/BedrockPluginLoader

CMD ["./BedrockPluginLoader"]