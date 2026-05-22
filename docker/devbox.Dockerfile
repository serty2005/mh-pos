FROM mcr.microsoft.com/playwright:v1.59.1-noble

USER root

ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright \
    npm_config_cache=/home/pwuser/.npm \
    CI=1

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      git \
      procps \
      python3 \
      python-is-python3 \
      sqlite3 \
    && npx --yes playwright@1.59.1 install chromium --with-deps \
    && mkdir -p \
      /workspace/myhoreca-pos/pos-ui/node_modules \
      /workspace/myhoreca-pos/cloud-ui/node_modules \
      /home/pwuser/.npm \
    && chown -R pwuser:pwuser /workspace /home/pwuser/.npm /ms-playwright \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace/myhoreca-pos

USER pwuser

CMD ["bash", "-lc", "sleep infinity"]
