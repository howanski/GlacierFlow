FROM ubuntu:26.04

ARG GF_USER_ID=1000
ARG GF_PI_EXTRA_PACKAGES=""

# ubuntu already ships an 'ubuntu' user at uid 1000. If GF_USER_ID
# collides with an existing user, rename it to pi_dev and move its home dir
# to /home/pi_dev (used by compose mounts). Otherwise create a fresh user.
RUN set -eux; \
    if getent passwd "${GF_USER_ID}" >/dev/null; then \
        EXISTING_USER="$(getent passwd "${GF_USER_ID}" | cut -d: -f1)"; \
        usermod -l pi_dev "${EXISTING_USER}"; \
        usermod -d /home/pi_dev -m pi_dev; \
    else \
        adduser --disabled-password --uid "${GF_USER_ID}" pi_dev; \
    fi

# BASIC REQUIREMENTS
RUN apt-get update && \
    apt-get install -y --no-install-recommends bash curl tmux git vim nodejs npm ttyd && \
    rm -rf /var/lib/apt/lists/*

# USER-DEFINED PACKAGES (space-separated, from .env via compose build-arg)
RUN if [ -n "$GF_PI_EXTRA_PACKAGES" ]; then \
        apt-get update && \
        apt-get install -y --no-install-recommends $GF_PI_EXTRA_PACKAGES && \
        rm -rf /var/lib/apt/lists/*; \
    fi

USER pi_dev

CMD ["/pi_dev/entrypoint"]
