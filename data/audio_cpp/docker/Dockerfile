FROM ubuntu:26.04

RUN apt update

RUN apt install -y libgomp1 mesa-vulkan-drivers

WORKDIR /app/
ENTRYPOINT ["/app/audiocpp_server"]
