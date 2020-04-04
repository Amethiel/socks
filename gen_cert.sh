openssl rand -writerand ~/.rnd
openssl genrsa -rand ~/.rnd -out server.key 2048
openssl req -new -x509 -key server.key -sha256 -new -nodes -out server.pem -rand ~/.rnd -days 3650 -subj="/CN=focusworks.net"
rm ~/.rnd

openssl rand -writerand ~/.rnd
openssl genrsa -rand ~/.rnd -out client.key 2048
openssl req -new -x509 -key client.key -sha256 -new -nodes -out client.pem -rand ~/.rnd -days 3650 -subj="/CN=focusworks.net"
rm ~/.rnd
