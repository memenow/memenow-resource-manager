FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND noninteractive

WORKDIR /app

COPY memenow-resource-manager /app/memenow-resource-manager

RUN apt-get update && apt-get upgrade -y && apt install -y git wget curl htop net-tools\
    git clone https://github.com/memenow/stable-diffusion-webui-on-k8s.git . \
    chmod +x memenow-resource-manager

EXPOSE 8080

CMD ["/app/memenow-resource-manager"]
