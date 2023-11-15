package mittwaldv2

type ClientBuilder interface {
	Project() *ProjectClient
	App() AppClient
	Database() *DatabaseClient
}

type clientBuilder struct {
	internalClient ClientWithResponsesInterface
}

func (b *clientBuilder) Project() *ProjectClient {
	return &ProjectClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) Database() *DatabaseClient {
	return &DatabaseClient{
		client: b.internalClient,
	}
}

func (b *clientBuilder) App() AppClient {
	return &appClient{
		client: b.internalClient,
	}
}
