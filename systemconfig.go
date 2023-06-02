package main

func DefaultSystem(system *System) *System {
	client := system.AddNode(ClientType)
	client.SetPosition(Position{
		X: 400,
		Y: 425,
	})

	loadBalancer := system.AddNode(LoadBalancerType)
	loadBalancer.SetPosition(Position{
		X: 700,
		Y: 425,
	})

	server1 := system.AddNode(ServerType)
	server1.SetPosition(Position{
		X: 1050,
		Y: 225,
	})

	server2 := system.AddNode(ServerType)
	server2.SetPosition(Position{
		X: 1050,
		Y: 425,
	})

	server3 := system.AddNode(ServerType)
	server3.SetPosition(Position{
		X: 1050,
		Y: 625,
	})

	system.AddEdge(client.GetID(), loadBalancer.GetID())
	system.AddEdge(loadBalancer.GetID(), server1.GetID())
	system.AddEdge(loadBalancer.GetID(), server2.GetID())
	system.AddEdge(loadBalancer.GetID(), server3.GetID())

	return system
}
