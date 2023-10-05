import json

def worker_node(id, config):
    return f"""worker_{id}:
    build:
      context: .
      dockerfile: deploy/Dockerfile.server
    depends_on:
      master:
        condition: service_healthy # wait for master gRPC server to start
    environment:
      - "HOST=worker_{id}"
      - "RABBIT_HOST=rabbitmq"
      - "RABBIT_USER={config['rabbit_user']}"
      - "RABBIT_PASSWORD={config['rabbit_password']}"
      - "HEALTH_CHECK={config['health_check']}"
      - "PORT={config['grpc_port']}"
      - "MASTER=master:{config['grpc_port']}"
      - "API_PORT={config['api_port']}"
"""

config_file = open("config.json")
config = json.load(config_file)

compose_str = f"""services:
  client:
    build:
      context: .
      dockerfile: deploy/Dockerfile.client
    ports:
      - 80:80
    environment:
      - "HOST=client"
      - "RPC_PORT={config['grpc_port']}"
  rabbitmq:
    image: rabbitmq:management-alpine
    ports:
      - "15672:15672"
    environment:
      - "RABBITMQ_DEFAULT_USER={config['rabbit_user']}"
      - "RABBITMQ_DEFAULT_PASS={config['rabbit_password']}"
    healthcheck:
      test: [ "CMD", "rabbitmqctl", "status" ]
      interval: 10s
      timeout: 5s
      retries: 5
  master:
    build:
      context: .
      dockerfile: deploy/Dockerfile.server
    depends_on:
      rabbitmq:
        condition: service_healthy # wait for RabbitMQ to completely start
    environment:
      - "HOST=master"
      - "MASTER=master:{config['grpc_port']}"
      - "RABBIT_HOST=rabbitmq"
      - "RABBIT_USER={config['rabbit_user']}"
      - "RABBIT_PASSWORD={config['rabbit_password']}"
      - "PORT={config['grpc_port']}"
      - "API_PORT={config['api_port']}"
      - "HEALTH_CHECK={config['health_check']}"
    healthcheck:
      test: [ "CMD", "nc", "-z", "-w3", "localhost", "{config['grpc_port']}" ]
      interval: 10s
      timeout: 2s
      retries: 5
"""

for i in range(config['workers']):
    worker = worker_node(i, config)
    compose_str = f"""{compose_str}
  {worker}
    """

compose_file = open("compose.yaml", "w")
compose_file.write(compose_str)
compose_file.close()
