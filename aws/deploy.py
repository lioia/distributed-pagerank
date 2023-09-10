import subprocess
import json
import os
import time

def run_command(command):
    process = subprocess.Popen(
        command, 
        shell=True, 
        stdout=subprocess.PIPE, 
        stderr=subprocess.PIPE, 
        text=True
    )
    if process.stdout is not None:
        for line in process.stdout:
            print(line, end='')
    return_code = process.wait()
    if return_code != 0:
        print(f"Command {command} failed")
        exit(return_code)

key_pem = input("Please enter the path to key.pem: ")
os.chdir("aws")

print("Creating AWS EC2 instances using Terraform")
run_command("terraform init")
run_command("terraform plan")
ok = input("Do you want to continue? [y/n] ").lower()
if ok == "n" or ok == "no":
    print("Plan was not approved")
    exit(1)
run_command("terraform apply -auto-approve")
run_command("terraform output -json > tf.json")
print("AWS instances created correctly")
print("Waiting 30 sec for instances to start")
time.sleep(30)
print("Deploying using Ansible")
json_file = open("tf.json")
json_file = open("tf.json")
data = json.load(json_file)
public_master = data["dp-master-host"]["value"]
private_mq_host = data["dp-mq-host"]["value"]
mq_user = data["dp-mq-user"]["value"]
mq_password = data["dp-mq-password"]["value"]
public_mq_host = data["dp-mq-public-host"]["value"]
public_workers_hosts = data["dp-workers-hosts"]["value"]

print("Deploying RabbitMQ")
run_command(f"./mq.sh {key_pem} {public_mq_host} {mq_user} {mq_password}")
service = f'''[Unit]
Description=distributed-pagerank application

[Service]
Environment=PORT=1234
Environment=API_PORT=5678
Environment=HEALTH_CHECK=3000
Environment=MASTER={public_master}
Environment=RABBIT_HOST={private_mq_host}
Environment=RABBIT_USER={mq_user}
Environment=RABBIT_PASSWORD={mq_password}
Environment=NODE_LOG=false
Environment=SERVER_LOG=false
Type=simple
WorkingDirectory=/home/ec2-user/dp
ExecStart=/home/ec2-user/dp/build/node
ExecStop=/bin/kill -TERM $MAINPID

[Install]
WantedBy=multi-user.target
'''
service_file = open("dp.service", "w")
service_file.write(service)
print("Deploying Master")
run_command(f"./node.sh {key_pem} {public_master} {public_master} {private_mq_host} {mq_user} {mq_password}")
for worker in public_workers_hosts:
    print(f"Deploying Worker {worker}")
    run_command(f"./node.sh {key_pem} {worker} {public_master} {private_mq_host} {mq_user} {mq_password}")

print("Removing temp files")
os.remove("tf.json")
os.remove("dp.service")
print(f"Correctly deployed application. You can contact it at {public_master}")
