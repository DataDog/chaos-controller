FROM ubuntu:focal as client

COPY built_go_client /usr/local/bin/dogfood_tester

ENTRYPOINT [ "/usr/local/bin/dogfood_tester" ]
