FROM nginx:1.28

RUN apt-get update && \
    apt-get install -y nginx-module-njs && \
    rm -rf /var/lib/apt/lists/*