package mittwaldv2

type ClientBuilder interface {
	Project() ProjectClient
	App() AppClient
	Database() DatabaseClient
	Cronjob() CronjobClient
	Domain() DomainClient
	User() UserClient
}

type clientBuilder struct {
	internalClient ClientWithResponsesInterface
}

func (b *clientBuilder) Project() ProjectClient {
	return &projectClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) Database() DatabaseClient {
	return &databaseClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) App() AppClient {
	return &appClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) Cronjob() CronjobClient {
	return &cronjobClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) Domain() DomainClient {
	return &domainClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) User() UserClient {
	return &userClient{
		client: b.internalClient,
	}
}
