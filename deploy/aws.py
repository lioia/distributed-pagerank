import threading
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

def service(host):
    service = f"""[Unit]
Description=distributed-pagerank application

[Service]
Environment=PORT=1234
Environment=API_PORT=5678
Environment=HEALTH_CHECK=3000
Environment=HOST={host}
Environment=MASTER={private_master}:1234
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
    """
    service_file = open(f"dp.service_{host}", "w")
    service_file.write(service)
    service_file.close()

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
print("Deploying using custom scripts")
json_file = open("tf.json")
data = json.load(json_file)

private_master = data["dp-master-host-private"]["value"]
public_master = data["dp-master-host-public"]["value"]

private_mq_host = data["dp-mq-host-private"]["value"]
public_mq_host = data["dp-mq-host-public"]["value"]
mq_user = data["dp-mq-user"]["value"]
mq_password = data["dp-mq-password"]["value"]

public_workers_hosts = data["dp-workers-hosts-public"]["value"]
private_workers_hosts = data["dp-workers-hosts-private"]["value"]

public_client_host = data["dp-client-host-public"]["value"]
private_client_host = data["dp-client-host-private"]["value"]

threads: list[threading.Thread] = []

print("Deploying RabbitMQ")
t = threading.Thread(target=run_command, args=(f"./mq.sh {key_pem} {public_mq_host} {mq_user} {mq_password}",))
t.start()
threads.append(t)

print("Deploying Master")
service(private_master)
t = threading.Thread(target=run_command, args=(f"./node.sh {key_pem} {public_master} {private_master}",))
t.start()
threads.append(t)

for i in range(len(public_workers_hosts)):
    worker = public_workers_hosts[i]
    private_worker = private_workers_hosts[i]
    print(f"Deploying Worker {worker}")
    service(private_worker)
    t = threading.Thread(target=run_command, args=(f"./node.sh {key_pem} {worker} {private_worker}",))
    threads.append(t)
    t.start()

client_service = f'''[Unit]
Description=distributed-pagerank application

[Service]
Environment=HOST={private_client_host}
Environment=RPC_PORT=1234
Type=simple
WorkingDirectory=/home/ec2-user/dp
ExecStart=/home/ec2-user/dp/build/client
ExecStop=/bin/kill -TERM $MAINPID

[Install]
WantedBy=multi-user.target
'''
client_service_file = open("dp-client.service", "w")
client_service_file.write(client_service)
client_service_file.close()
print("Deploying Client")
t = threading.Thread(target=run_command, args=(f"./client.sh {key_pem} {public_client_host}",))
t.start()
threads.append(t)

# Wait for all threads to finish
for t in threads:
    t.join()

print("Removing temp files")
os.remove("tf.json")
os.remove(f"dp.service_{private_master}")
for worker in private_workers_hosts:
    os.remove(f"dp.service_{worker}")
os.remove("dp-client.service")
print(f"Correctly deployed application. You can contact it at {public_client_host} (API at: {private_master}:5678)")
