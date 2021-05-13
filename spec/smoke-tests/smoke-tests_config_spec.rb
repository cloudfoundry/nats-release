require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'config.json.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) do
        { }
      end

      let(:links) do
        [
          Link.new(
            name: 'nats',
            instances: [
              LinkInstance.new(id: 'meowmeowmeow'),
            ],
            properties: {
              'nats-tls' => {
                'user' => 'my-tls-user',
                'password' => 'my-tls-password',
                'hostname' => 'my-host',
                'port' => 4222,
                'cluster_port' => 4223,
                'http' => '0.0.0.0:0'
              }
            }
          ),
          Link.new(
            name: 'nats-tls',
            instances: [
              LinkInstance.new(id: 'meow-tls'),
            ],
            properties: {
              'nats' => {
                'user' => 'my-tls-user',
                'password' => 'my-tls-password',
                'hostname' => 'my-tls-host',
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

      describe 'smoke-tests job' do

        let(:job) {release.job('smoke-tests')}

        describe 'bin/config.json' do
          let(:template) { job.template('bin/config.json') }

          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
expected_template =  {
  :NonTLS => {
    :Hosts => ["my-host"],
    :Port => 4222,
    :User => "my-user",
    :Password => "my-password",
  },
  :TLS => {
    :Hosts => ["my-tls-host"],
    :Port => 4224,
    :User => "my-tls-user",
    :Password => "my-tls-password",
    :Certificate => "/var/vcap/jobs/smoke-tests/config/client_tls/certificate.pem",
    :Ca => "/var/vcap/jobs/smoke-tests/config/client_tls/ca.pem",
    :PrivateKey => "/var/vcap/jobs/smoke-tests/config/client_tls/private_key.pem",
  },
}.to_json
            expect(rendered_template).to include(expected_template)
          end

          # describe 'nats machine ips are provided' do
          #   before do
          #     merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
          #   end

          #   it 'renders the template with the provided manifest properties' do
          #     rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
# expected_template =  %{
# net: "10.0.0.1"
# port: 4222
# prof_port: 0
# http: "0.0.0.0:0"
# write_deadline: "2s"

# debug: false
# trace: false
# logtime: true

# authorization \{
  # user: "my-user"
  # password: "my-password"
  # timeout: 15
# \}

# cluster \{
  # no_advertise: true
  # host: "10.0.0.1"
  # port: 4223

  # authorization \{
    # user: "my-user"
    # password: "my-password"
    # timeout: 15
  # \}

  
  # tls \{
    # ca_file: "/var/vcap/jobs/nats/config/internal_tls/ca.pem"
    # cert_file: "/var/vcap/jobs/nats/config/internal_tls/certificate.pem"
    # key_file: "/var/vcap/jobs/nats/config/internal_tls/private_key.pem"
    # cipher_suites: \[
      # "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
      # "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    # \]
    # curve_preferences: \[
      # "CurveP384"
    # \]
    # timeout: 5 # seconds
    # verify: true
  # \}
  

  # routes = \[
    
    # nats-route://my-user:my-password@192.0.0.1:4223
    
    # nats-route://my-user:my-password@198.5.4.3:4223
    
  # \]
# \}
# }

          #     expect(rendered_template).to include(expected_template)
          #   end
          # end
          # describe 'nats machine ips and tls_cluster_port are provided' do
          #   before do
          #     merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
          #     merged_manifest_properties['nats']['tls_cluster_port'] = '4225'
          #   end

          #   it 'renders the template with the provided manifest properties' do
          #     rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
# expected_template =  %{
# net: "10.0.0.1"
# port: 4222
# prof_port: 0
# http: "0.0.0.0:0"
# write_deadline: "2s"

# debug: false
# trace: false
# logtime: true

# authorization \{
  # user: "my-user"
  # password: "my-password"
  # timeout: 15
# \}

# cluster \{
  # no_advertise: true
  # host: "10.0.0.1"
  # port: 4223

  # authorization \{
    # user: "my-user"
    # password: "my-password"
    # timeout: 15
  # \}

  
  # tls \{
    # ca_file: "/var/vcap/jobs/nats/config/internal_tls/ca.pem"
    # cert_file: "/var/vcap/jobs/nats/config/internal_tls/certificate.pem"
    # key_file: "/var/vcap/jobs/nats/config/internal_tls/private_key.pem"
    # cipher_suites: \[
      # "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
      # "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
    # \]
    # curve_preferences: \[
      # "CurveP384"
    # \]
    # timeout: 5 # seconds
    # verify: true
  # \}
  

  # routes = \[
    
    # nats-route://my-user:my-password@192.0.0.1:4223
    
    # nats-route://my-user:my-password@198.5.4.3:4223
    
    # nats-route://my-user:my-password@192.0.0.1:4225
    
    # nats-route://my-user:my-password@198.5.4.3:4225
    
  # \]
# \}
# }

          #     expect(rendered_template).to include(expected_template)
          #   end
          # end
        end
      end
    end
  end
end
