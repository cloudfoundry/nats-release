require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'nats.conf.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) do
        {
          'nats' => {
            'user' => 'my-user',
            'password' => 'my-password',
            'hostname' => 'my-host',
            'port' => 4222,
            'cluster_port' => 4223,
            'authorization_timeout' => 15,
            'machines' => ['192.168.1.1', '192.168.1.2'],
            'debug' => false,
            'trace' => false,
            'monitor_port' => 0,
            'prof_port' => 0,
            'internal' => {
              'tls' => {
                'enabled' => false,
                'ca' => 'internal-tls-ca',
                'certificate' => 'internal-tls-cert',
                'private_key' => 'internal-tls-key'
              }
            }
          }
        }
      end

      let(:links) do
        [
          Link.new(
            name: 'nats',
            instances: [LinkInstance.new()],
            properties: {
              'user' => 'my-user',
              'password' => 'my-password',
              'hostname' => 'my-host',
              'port' => 4222,
              'cluster_port' => 4223,
              'monitor_port' => 0
            }
          )
        ]
      end

      let(:spec) do
        {
          'address' => '10.0.0.1'
        }
      end

      describe 'nats job' do

        let(:job) {release.job('nats')}

        describe 'config/nats.conf' do
          let(:template) { job.template('config/nats.conf') }

          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
expected_template =  %{
net: "10.0.0.1"
port: 4222
prof_port: 0
monitor_port: 0

debug: false
trace: false
logtime: true

authorization \{
  user: "my-user"
  password: "my-password"
  timeout: 15
\}

cluster \{
  host: "10.0.0.1"
  port: 4223

  authorization \{
    user: "my-user"
    password: "my-password"
    timeout: 15
  \}

  

  routes = \[
    
    nats-route://my-user:my-password@my-host:4223
    
  \]
\}
}
            expect(rendered_template).to include(expected_template)
          end
        end
      end
    end
  end
end
