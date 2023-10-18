import json

config_file = open("config.json")
config = json.load(config_file)

base_env = f"""HOST=localhost
RABBIT_HOST=localhost
API_PORT={config['api_port']}
RPC_PORT={config['grpc_port']}
PORT={config['grpc_port']+1}
MASTER=localhost:{config['grpc_port']+1}
HEALTH_CHECK={config['health_check']}
RABBIT_USER={config['rabbit_user']}
RABBIT_PASSWORD={config['rabbit_password']}
WEB_PORT=8080
NODE_LOG={config['node_log']}
SERVER_LOG={config['server_log']}
"""

env_file = open(".env", "w")
env_file.write(base_env)
env_file.close()

worker_command = f""

for i in range(config['workers']):
    worker_command = f"{worker_command}PORT={config['grpc_port'] + 2 + i} ./build/server & "

print("You can run locally by executing the following commands:\nRabbitMQ:")
print(f"\tdocker run -it --rm --name rabbitmq -p 5672:5672 -e RABBITMQ_DEFAULT_USER={config['rabbit_user']} -e RABBITMQ_DEFAULT_PASS={config['rabbit_password']} rabbitmq:3.12-alpine")
print("Master\n\t./build/server")
print(f"{config['workers']} workers")
print(f"\t{worker_command}")
print(f"Web client")
print(f"\t./build/client")

