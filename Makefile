NAME = redditbot

build:
	docker build --tag $(NAME) . 

# Creates a container
# Runs a command
create:
	docker run --interactive --tty --entrypoint=/bin/bash --env-file reddit-wikipediaposter.env  $(NAME) -i
