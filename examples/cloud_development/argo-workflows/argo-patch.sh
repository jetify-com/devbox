# Patch so we don't need to login through the UI, for development purposes. See https://argoproj.github.io/argo-workflows/quick-start/ for details.

kubectl patch deployment \
  argo-server \
  --namespace=argo \
  --type='json' \
  -p='[{"op": "replace", "path": "/spec/template/spec/containers/0/args", "value": [
  "server",
  "--auth-mode=server"
]}]'