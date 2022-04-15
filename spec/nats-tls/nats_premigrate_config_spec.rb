require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'nats-premigrate.conf.erb' do
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
            'machines' => nil,
            'no_advertise' => false,
            'debug' => false,
            'trace' => false,
            'http' => '0.0.0.0:0',
            'prof_port' => 0,
            'external' => {
              'tls' => {
                'ca' => 'external-tls-ca',
                'certificate' => 'external-tls-cert',
                'private_key' => 'external-tls-key'
              }
            },
            'internal' => {
              'tls' => {
                'enabled' => true,
                'ca' => 'internal-tls-ca',
                'certificate' => 'internal-tls-cert',
                'private_key' => 'internal-tls-key'
              }
            },
            'client' => {
              'tls' => {
                'certificate' => 'client-tls-cert',
                'private_key' => 'client-tls-key'
              }
            }
          }
        }
      end

      let(:links) do
        [
          Link.new(
            name: 'nats',
            instances: [
              LinkInstance.new(id: 'abc123'),
              LinkInstance.new(id: 'der456')
            ],
            properties: {
              'nats' => {
                'user' => 'my-user',
                'password' => 'my-password',
                'hostname' => 'my-host',
                'port' => 4224,
                'cluster_port' => 4225,
                'http' => '0.0.0.0:0'
              }
            }
          )
        ]
      end

      let(:spec) do
        {
          'address' => '10.0.0.1'
        }
      end

      describe 'nats-tls job' do

        let(:job) {release.job('nats-tls')}

        describe 'config/pre-migrate.conf' do
          let(:template) { job.template('config/pre-migrate.conf') }

          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
           
            expected_template =  %{
               "nats_machines": [ "abc123.nats.my-host:4224", "def456.nats.my-host:4244" ],
               "nats_bpm_config_path": "/var/vcap/jobs/nats-tls/config/bpm.yml",
               "v1_bpm_config_path": "/var/vcap/jobs/nats-tls/config/bpm.v1.yml"
            }
            expect(rendered_template).to include(expected_template)


          end
        end
      end
    end
  end
end
