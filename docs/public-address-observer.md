# Public libp2p address observer

The Discovery service runs a small libp2p host that observes a GoChat peer's UDP/QUIC source address. This is address-observation/control traffic only: the observer does not implement the chat protocol, carry chat content, or relay peer traffic.

## Configure Discovery on EC2

1. Set the GitHub repository variable `DISCOVERY_OBSERVER_ADVERTISE_ADDR` to the observer's public transport address, without `/p2p`, for example `/ip4/<EC2_PUBLIC_IPV4>/udp/4001/quic-v1`. Prefer an Elastic IP or stable DNS name.
2. Add an inbound **UDP 4001** rule to the EC2 security group. Compose maps this UDP port into Discovery. Do not open Redis to the internet.
3. Deploy the Discovery workflow. The observer's identity key is stored in the persistent `observer_data` Docker volume, so its Peer ID stays the same across normal container replacements and EC2 restarts. Do not remove that volume.
4. Confirm the observer is available with `curl http://<EC2_PUBLIC_IPV4>:8080/observer`. The response contains the complete multiaddress, including `/p2p/<PeerID>`; clients retrieve it from Discovery automatically.

## GoChat client flow

The client calls `GET /observer` on Discovery to retrieve the observer's complete multiaddress, including its stable Peer ID. No observer address variable is required in the client's `.env.chat`; `DISCOVERY_URL` is sufficient.

## How observation works

After getting the multiaddress, the client dials the observer over authenticated libp2p QUIC/UDP. The observer sees the source IP and UDP port of that connection and associates it with the authenticated GoChat Peer ID. The client then opens the small `/gochat/address-observation/1.0.0` control protocol over the same libp2p connection and asks for its observed address. It publishes the result alongside its other candidates through the normal Discovery heartbeat. `GET /observer` returns public connection metadata only; it does not carry chat messages.

The observed endpoint is a candidate, not a guarantee: NATs can map different external ports for different remote destinations, and a firewall can still block inbound traffic. The later direct-dial/hole-punch flow must validate connectivity. Discovery stores address metadata only; neither Discovery nor the observer stores or forwards chat messages.
