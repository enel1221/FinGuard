# Bake dex dev config into image to avoid bind-mount issues on Docker Desktop for Mac
FROM dexidp/dex:v2.41.1
COPY configs/dex.dev.yaml /etc/dex/config.docker.yaml
# default CMD/ENTRYPOINT from base image
