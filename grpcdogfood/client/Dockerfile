FROM ubuntu:focal as client

COPY built_go_client /usr/local/bin/dogfood_client

EXPOSE 51002

ENTRYPOINT [ "/usr/local/bin/dogfood_client" ]
