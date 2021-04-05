#!/usr/bin/env python

from diagrams import Diagram, Cluster
from diagrams.k8s.compute import Deployment
from diagrams.onprem.compute import Server
from diagrams.onprem.network import Internet
from diagrams.onprem.queue import RabbitMQ

with Diagram("mailer", show=False, direction="TB"):
    with Cluster("github.com/ViBiOh"):
        mailer = Deployment("mailer")
        mjml = Deployment("mjml-api")

    [Internet("HTTP"), RabbitMQ("AMQP")] >> mailer >> [
        mjml,
        Server("SMTP"),
    ]
