require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'healthcheck-config.json.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) do
        {
          'nats' => {
            'user' => 'my-user',
            'password' => 'my-password',
            'hostname' => 'my-host',
            'port' => 4224,
          }
        }
      end

      let(:spec) do
        {
          'address' => '10.0.0.1'
        }
      end

      describe 'nats-tls job' do
        let(:job) { release.job('nats-tls') }
        let(:template) { job.template('config/healthcheck-config.json') }

        it 'renders the template with the provided manifest properties' do
          rendered_template = JSON.parse(template.render(merged_manifest_properties, spec: spec))
          expected_template = {
            'address'            => '10.0.0.1',
            'port'               => '4224',
            'server_ca'          => '/var/vcap/jobs/nats-tls/config/external_tls/ca.pem',
            'server_hostname'    => 'my-host',
            'client_certificate' => '/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem',
            'client_private_key' => '/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem',
            'user'               => 'my-user',
            'password'           => 'my-password'
          }
          expect(rendered_template).to eq(expected_template)
        end

        describe 'when user and password are not provided' do
          before do
            merged_manifest_properties['nats'].delete('user')
            merged_manifest_properties['nats'].delete('password')
          end

          it 'renders the template without user and password' do
            rendered_template = JSON.parse(template.render(merged_manifest_properties, spec: spec))
            expected_template = {
              'address'            => '10.0.0.1',
              'port'               => '4224',
              'server_ca'          => '/var/vcap/jobs/nats-tls/config/external_tls/ca.pem',
              'server_hostname'    => 'my-host',
              'client_certificate' => '/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem',
              'client_private_key' => '/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem'
            }
            expect(rendered_template).to eq(expected_template)
            expect(rendered_template).not_to have_key('user')
            expect(rendered_template).not_to have_key('password')
          end
        end
      end
    end
  end
end
