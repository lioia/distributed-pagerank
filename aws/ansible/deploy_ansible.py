import subprocess
import time
import os
import json

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

print("WARN: This does not seem to work (ansible hangs when adding new user to RabbitMQ)")
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
print("-------------------------------")
print("Waiting 30 sec for instances to start")
time.sleep(30)
print("Deploying using Ansible")
json_file = open("tf.json")
data = json.load(json_file)
vars = {
    "DP_MASTER": data["dp-master-host"]["value"],
    "DP_RABBIT_HOST" : data["dp-mq-host"]["value"],
    "DP_RABBIT_USER" : data["dp-mq-user"]["value"],
    "DP_RABBIT_PASSWORD" : data["dp-mq-password"]["value"]
}
vars_file = open("vars.json", "w")
json.dump(vars, vars_file)
print("\tDeploying RabbitMQ")
run_command(f"ANSIBLE_SSH_ARGS='-o StrictHostKeyChecking=no' ansible-playbook -i mq.aws_ec2.yml --private-key={key_pem} -u ec2-user mq.yaml")
print("\tDeploying application")
run_command(f"ANSIBLE_SSH_ARGS='-o StrictHostKeyChecking=no' ansible-playbook -i node.aws_ec2.yml --private-key={key_pem} -u ec2-user node.yaml")
print("-------------------------------")
print("Remove output files")
os.remove("vars.json")
os.remove("tf.json")
print("Correctly deployed application")
print("-------------------------------")
