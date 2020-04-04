#!/bin/bash

# openssl rand -writerand ~/.rnd
# openssl genrsa -rand ~/.rnd -out server.key 2048
# openssl req -new -x509 -key server.key -sha256 -new -nodes -out server.pem -rand ~/.rnd -days 3650 -subj="/CN=focusworks.net"
# rm ~/.rnd
# 
# openssl rand -writerand ~/.rnd
# openssl genrsa -rand ~/.rnd -out client.key 2048
# openssl req -new -x509 -key client.key -sha256 -new -nodes -out client.pem -rand ~/.rnd -days 3650 -subj="/CN=focusworks.net"
# rm ~/.rnd

function gen_ca() {
    echo gen_ca $@ ...

    openssl genrsa -out ca.key 2048 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate CA createserial.
        return 11
    fi

    openssl req -new -key ca.key -out ca.csr -subj "/CN=focusworks.net" 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate CA createserial.
        return 12
    fi

    openssl x509 -req -days 3650 -sha1 -extensions v3_ca -signkey ca.key -in ca.csr -out ca.pem 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate CA createserial.
        return 13
    fi

    echo gen_ca $@ ... done.
}

function gen_server_cert() {
    echo gen_server_cert $@ ...

    openssl genrsa -out server.key 2048 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate server createserial.
        return 21
    fi

    openssl req -new -key server.key -out server.csr -subj "/CN=proxy.focusworks.net" 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate server createserial.
        return 22
    fi

    openssl x509 -req -days 3650 -sha1 -extensions v3_req -CA ca.pem -CAkey ca.key -CAcreateserial -in server.csr -out server.pem 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate server createserial.
        return 23
    fi

    echo gen_server_cert $@ ... done.
}

function gen_client_cert() {
    echo gen_client_cert $@ ...
    client_name=$1

    openssl genrsa -out ${client_name}.key 2048 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate client createserial.
        return 31
    fi

    openssl req -new -key ${client_name}.key -out ${client_name}.csr -subj "/CN=${client_name}.focusworks.net" 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate client createserial.
        return 32
    fi
    openssl x509 -req -days 3650 -sha1 -extensions v3_req -CA ca.pem -CAkey ca.key -CAcreateserial -in ${client_name}.csr -out ${client_name}.pem 2>/dev/null
    if [ $? -ne 0 ]; then
        echo FAILED to generate client createserial.
        return 33
    fi

    echo gen_client_cert $@ ... done.
}

gen_ca
gen_server_cert
gen_client_cert client1
