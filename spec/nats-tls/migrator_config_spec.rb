require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'migrator-config.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) do
        {
          'nats' => {
            'internal' => {
              'tls' => {
                'enabled' => true,
              }
            },
          }
        }
      end

      let(:links) do
        [
          Link.new(
            name: 'nats-tls',
            instances: [
              LinkInstance.new(id: 'abc1234'),
              LinkInstance.new(id: 'def456'),
              LinkInstance.new(id: 'bbc790')
            ],
            properties: {
              'nats' => {
                'hostname' => 'nats.service.cf.internal',
                'port' => 4224,
              }
            }
          )
        ]
      end

      let(:spec) do
        {
          'bootstrap' => true,
          'id' => "bbc790"
        }
      end

      describe 'nats-tls job' do

        let(:job) {release.job('nats-tls')}

        describe 'config/migrator-config.json' do
          let(:template) { job.template('config/migrator-config.json') }

          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
expected_template =  %{
{
    "bootstrap": true,
    "address": "bbc790.nats.service.cf.internal",
    "nats_instances": [ "abc1234.nats.service.cf.internal:4224", "def456.nats.service.cf.internal:4224", "bbc790.nats.service.cf.internal:4224" ],
    "nats_port": 4224,
    "nats_migrate_port": 4243,
    "nats_v1_bin_path": "/var/vcap/packages/gnatsd/bin/gnatsd",
    "nats_v2_bin_path": "/var/vcap/packages/nats-server/bin/nats-server",
    "nats_config_path": "/var/vcap/jobs/nats-tls/config/nats-tls.conf",
    "nats_migrate_servers": [ "https://abc1234.nats.service.cf.internal:4243", "https://def456.nats.service.cf.internal:4243", "https://bbc790.nats.service.cf.internal:4243" ],
    "nats_internal_tls_enabled": true,
    "nats_migrate_server_ca_file": "/var/vcap/jobs/nats-tls/config/external_tls/ca.pem",
    "nats_migrate_server_cert_file": "/var/vcap/jobs/nats-tls/config/external_tls/certificate.pem",
    "nats_migrate_server_key_file": "/var/vcap/jobs/nats-tls/config/external_tls/private_key.pem",
    "nats_migrate_client_ca_file": "/var/vcap/jobs/nats-tls/config/client_tls/ca.pem",
    "nats_migrate_client_cert_file": "/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem",
    "nats_migrate_client_key_file": "/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem"
}
}
            expect(rendered_template).to include(expected_template)
          end
        end
      end
    end
  end
end
