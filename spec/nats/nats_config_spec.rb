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
            'machines' => nil,
            'no_advertise' => true,
            'debug' => false,
            'trace' => false,
            'http' => '0.0.0.0:0',
            'prof_port' => 0,
            'internal' => {
              'tls' => {
                'enabled' => true,
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
            instances: [
              LinkInstance.new(id: 'meowmeowmeow'),
              LinkInstance.new(id: 'a-b-c-d')
            ],
            properties: {
              'nats' => {
                'user' => 'my-user',
                'password' => 'my-password',
                'hostname' => 'my-host',
                'port' => 4222,
                'cluster_port' => 4223,
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

      describe 'nats job' do

        let(:job) {release.job('nats')}

        describe 'config/nats.conf' do
          let(:template) { job.template('config/nats.conf') }

          it 'renders the template with the provided manifest properties' do
            rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
            expect(rendered_hash).to eq(expected_hash)
          end

          describe 'nats machine ips are provided' do
            before do
              merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
              expected_hash['cluster']['routes'] =
                [
                  'nats-route://my-user:my-password@192.0.0.1:4223',
                  'nats-route://my-user:my-password@198.5.4.3:4223'
                ]
            end

            it 'renders the template with the provided manifest properties' do
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end

          describe 'nats machine ips and tls_cluster_port are provided' do
            before do
              merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
              merged_manifest_properties['nats']['tls_cluster_port'] = '4225'
              expected_hash['cluster']['routes'] =
                [
                  'nats-route://my-user:my-password@192.0.0.1:4223',
                  'nats-route://my-user:my-password@198.5.4.3:4223',
                  'nats-route://my-user:my-password@192.0.0.1:4225',
                  'nats-route://my-user:my-password@198.5.4.3:4225',
                ]
            end

            it 'renders the template with the provided manifest properties' do
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end
        end
      end
    end
  end
end
