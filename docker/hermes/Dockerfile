FROM ubuntu:26.04

RUN apt update

RUN apt install -y bash curl ttyd tmux git vim python3 nodejs npm ripgrep ffmpeg

USER ubuntu

RUN curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash

CMD ["/init-data/entrypoint"]
