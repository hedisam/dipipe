FROM golang:1.16.5-alpine
ARG plugin_name
ENV env_plugin_name "$plugin_name.so"
RUN apk add build-base
RUN mkdir -p /app/plugin
WORKDIR /app
COPY . .
RUN GOOS=linux go build -buildmode=plugin -o ${plugin_name}.so .
CMD ["sh", "-c", "cp $env_plugin_name plugin/$env_plugin_name"]
