# Public libp2p address observer

The Discovery service runs a small libp2p host that observes a GoChat peer's UDP/QUIC source address. This is address-observation/control traffic only: the observer does not implement the chat protocol, carry chat content, or relay peer traffic.

## Configure Discovery on EC2

1. Set the GitHub repository variable `DISCOVERY_OBSERVER_ADVERTISE_ADDR` to the observer's public transport address, without `/p2p`, for example `/ip4/<EC2_PUBLIC_IPV4>/udp/4001/quic-v1`. Prefer an Elastic IP or stable DNS name.
2. Add an inbound **UDP 4001** rule to the EC2 security group. Compose maps this UDP port into Discovery. Do not open Redis to the internet.
3. Deploy the Discovery workflow. The observer's identity key is stored in the persistent `observer_data` Docker volume, so its Peer ID stays the same across normal container replacements and EC2 restarts. Do not remove that volume.
4. On EC2, run `sudo docker compose -f /home/ec2-user/gochat/docker-compose.yml logs discovery` and copy the complete multiaddress after `libp2p address observer ready at`. It includes the `/p2p/<PeerID>` component.

## Configure the GoChat client

Put that complete multiaddress in the client's `.env.chat` as `DISCOVERY_OBSERVER_PUBLIC_ADDR`, for example `/ip4/<EC2_PUBLIC_IPV4>/udp/4001/quic-v1/p2p/<OBSERVER_PEER_ID>`. The client uses this value directly; it does not call an HTTP `/observer` endpoint. The `.env.chat.example` documents this setting.

## How observation works

The client dials the configured observer over authenticated libp2p QUIC/UDP. The observer sees the source IP and UDP port of that connection and associates it with the authenticated GoChat Peer ID. The client then opens the small `/gochat/address-observation/1.0.0` control protocol over the same libp2p connection and asks for its observed address. It publishes the result alongside its other candidates through the normal Discovery heartbeat. `/observer` is not part of the HTTP API.

The observed endpoint is a candidate, not a guarantee: NATs can map different external ports for different remote destinations, and a firewall can still block inbound traffic. The later direct-dial/hole-punch flow must validate connectivity. Discovery stores address metadata only; neither Discovery nor the observer stores or forwards chat messages.
