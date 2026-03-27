make build-agent
make build-raw
scp iscsi_gui.raw root@10.0.1.233:/tmp/iscsi-gui.raw
ssh root@10.0.1.233 "sudo zpkg install /tmp/iscsi_gui.raw && sudo systemctl restart iscsi-agent"
