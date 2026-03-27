FROM golang:1.22

# Install Node.js + Claude Code
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs jq \
    && npm install -g @anthropic-ai/claude-code

WORKDIR /workspace

# Copy test repo
COPY repo/ .

# Copy AI config (empty for bare, skill for single, bundle for full)
COPY config/ .claude/

# Copy auth
COPY .env /root/.claude/.env

# Copy prompt and verification
COPY prompt.txt .
COPY verify.sh .

# Run: claude prompt → verify → output metrics
CMD ["bash", "-c", "\
    START=$(date +%s) && \
    claude -p \"$(cat prompt.txt)\" --output-format json > /tmp/claude_output.json 2>&1 && \
    END=$(date +%s) && \
    WALL_TIME=$((END - START)) && \
    bash verify.sh | jq --arg wt \"$WALL_TIME\" '. + {wall_time_seconds: ($wt | tonumber)}' \
"]
