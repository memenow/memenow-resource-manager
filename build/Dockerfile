FROM debian:bookworm-slim

WORKDIR /app

COPY memenow-resource-manager /app/memenow-resource-manager

RUN chmod +x memenow-resource-manager

EXPOSE 8080

CMD ["/app/memenow-resource-manager"]
