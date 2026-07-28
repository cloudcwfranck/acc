# Example Dockerfile that intentionally violates acc policies
# This demonstrates verification gating in action

FROM alpine:latest@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# VIOLATION: Running as root (no USER directive)
# This violates the no-root-user policy in .acc/policy/default.rego

RUN apk add --no-cache nginx

# Missing: No labels (triggers warning)

EXPOSE 8080

CMD ["nginx", "-g", "daemon off;"]
