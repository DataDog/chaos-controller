FROM ubuntu:focal as client

COPY built_go_client /usr/local/bin/dogfood_client

EXPOSE 50052

ENTRYPOINT [ "/usr/local/bin/dogfood_client" ]
