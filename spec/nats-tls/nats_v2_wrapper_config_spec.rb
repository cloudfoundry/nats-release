require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'nats-v2-wrapper-config.json.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) do
        {
          'nats' => {
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
        let(:template) { job.template('config/nats-v2-wrapper-config.json') }

        it 'renders the template with the provided manifest properties' do
          rendered_template = JSON.parse(template.render(merged_manifest_properties, spec: spec))
          expected_template = {
            "nats_v2_wrapper_port" => 4242,
            "nats_v2_wrapper_server_ca_file" => "/var/vcap/jobs/nats-tls/config/external_tls/ca.pem",
            "nats_v2_wrapper_server_cert_file" => "/var/vcap/jobs/nats-tls/config/external_tls/certificate.pem",
            "nats_v2_wrapper_server_key_file" => "/var/vcap/jobs/nats-tls/config/external_tls/private_key.pem",
            "nats_v2_bin_path" => "/var/vcap/packages/nats-server/bin/nats-server",
            "nats_config_path" => "/var/vcap/jobs/nats-tls/config/nats-tls.conf"
          }
          expect(rendered_template).to eq(expected_template)
        end
      end
    end
  end
end
