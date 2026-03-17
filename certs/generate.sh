#!/bin/bash
# Генерация самоподписанных TLS-сертификатов для разработки.
set -e

CERT_DIR="$(cd "$(dirname "$0")" && pwd)"

# CA сертификат
openssl req -x509 -newkey rsa:4096 -days 365 -nodes \
  -keyout "$CERT_DIR/ca-key.pem" \
  -out "$CERT_DIR/ca-cert.pem" \
  -subj "/CN=GophKeeper CA"

# Серверный ключ и CSR
openssl req -newkey rsa:4096 -nodes \
  -keyout "$CERT_DIR/server-key.pem" \
  -out "$CERT_DIR/server-req.pem" \
  -subj "/CN=localhost"

# Подписание серверного сертификата CA
openssl x509 -req -in "$CERT_DIR/server-req.pem" \
  -days 365 -CA "$CERT_DIR/ca-cert.pem" -CAkey "$CERT_DIR/ca-key.pem" \
  -CAcreateserial \
  -out "$CERT_DIR/server-cert.pem" \
  -extfile <(printf "subjectAltName=DNS:localhost,IP:127.0.0.1")

# Удаление промежуточных файлов
rm -f "$CERT_DIR/server-req.pem" "$CERT_DIR/ca-cert.srl"

echo "Сертификаты сгенерированы в $CERT_DIR"
echo "  CA:     ca-cert.pem"
echo "  Server: server-cert.pem + server-key.pem"
