require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'nats-tls.conf.erb' do
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
      let(:expected_hash) do
           {
             'net' => '10.0.0.1',
             'port' => 4222,
             'prof_port' => 0,
             'http' => '0.0.0.0:0',
             'write_deadline' => '2s',
             'debug' => false,
             'trace' => false,
             'logtime' => true,
             'authorization' => {
               'user' => "my-user",
               'password' => "my-password",
               'timeout' => 15,
             },
             'tls' => {
               'ca_file' => "/var/vcap/jobs/nats-tls/config/external_tls/ca.pem",
               'cert_file' => "/var/vcap/jobs/nats-tls/config/external_tls/certificate.pem",
               'key_file' => "/var/vcap/jobs/nats-tls/config/external_tls/private_key.pem",
               'cipher_suites' => [
                 "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
                 "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
               ],
               'curve_preferences' => [
                 "CurveP384",
               ],
               'timeout' => 5,
               'verify' => true,
             },
             'cluster' => {
               'no_advertise' => false,
               'host' => "10.0.0.1",
               'port' => 4223,
               'pool_size' => -1,
               'authorization' => {
                 'user' => "my-user",
                 'password' => "my-password",
                 'timeout' => 15,
               },
               'tls' => {
                 'ca_file' => "/var/vcap/jobs/nats-tls/config/internal_tls/ca.pem",
                 'cert_file' => "/var/vcap/jobs/nats-tls/config/internal_tls/certificate.pem",
                 'key_file' => "/var/vcap/jobs/nats-tls/config/internal_tls/private_key.pem",
                 'cipher_suites' => [
                   "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
                   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
                 ],
                 'curve_preferences' => [
                   "CurveP384",
                 ],
                 'timeout' => 5,
                 'verify' => true,
               },
               'routes' => [
                'nats-route://my-user:my-password@meowmeowmeow.my-host:4223',
                'nats-route://my-user:my-password@a-b-c-d.my-host:4223',
               ]
             },
             'no_sys_acc' => true,
           }
      end

      describe 'nats-tls job' do

        let(:job) {release.job('nats-tls')}

        describe 'config/nats.conf' do
          let(:template) { job.template('config/nats-tls.conf') }

          it 'renders the template with the provided manifest properties' do
            rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
            expect(rendered_hash).to eq(expected_hash)
          end

          describe 'when nats is disabled' do
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
                      'password' => 'nats-password',
                      'hostname' => 'nats-host',
                      'port' => 4222,
                      'cluster_port' => 4223,
                      'http' => '0.0.0.0:0',
                      'disable' => true
                    }
                  }
                )
              ]
            end
            it 'does not include nats in the routes' do
              expected_hash['cluster']['routes'] = []
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end

          describe 'nats machine ips are provided' do
            before do
              merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
              expected_hash['cluster']['routes'] = [
                'nats-route://my-user:my-password@192.0.0.1:4223',
                'nats-route://my-user:my-password@198.5.4.3:4223'
              ]
            end

            it 'renders the template with the provided manifest properties' do
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end

          describe 'nats machine ips and nontls_cluster_port are provided' do
            before do
              merged_manifest_properties['nats']['machines'] = ['192.0.0.1', '198.5.4.3']
              merged_manifest_properties['nats']['nontls_cluster_port'] = '4225'
              expected_hash['cluster']['routes'] = [
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

          describe 'password authentication is disabled' do
            before do
              merged_manifest_properties['nats']['user'] = nil
              merged_manifest_properties['nats']['password'] = nil
              expected_hash.delete('authorization')
              expected_hash['cluster'].delete('authorization')
            end

            it 'renders the template without password authentication properties' do
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end
          
          describe 'password authentication is disabled due to auth_required = false' do
            before do
              merged_manifest_properties['nats']['auth_required'] = false
              expected_hash.delete('authorization')
              expected_hash['cluster'].delete('authorization')
            end
  
            it 'renders the template without password authentication properties' do
              rendered_hash = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expect(rendered_hash).to eq(expected_hash)
            end
          end
        end

        describe 'config/bpm.yml' do
          let(:template) { job.template('config/bpm.yml') }

          it 'renders the template with the provided manifest properties' do
            rendered_template = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
            expected_template = {
              'processes' => [
                {
                  'name' => 'nats-tls-wrapper',
                  'limits' => {
                    'open_files' => 100000
                  },
                  'executable' => '/var/vcap/packages/nats-v2-migrate/bin/nats-wrapper',
                  'args' => ['--config-file', '/var/vcap/jobs/nats-tls/config/migrator-config.json']
                },
                {
                  'name' => 'healthcheck',
                  'executable' => '/var/vcap/packages/nats-tls-healthcheck/bin/nats-tls-healthcheck',
                  'args' => [
                    '--address',
                    '10.0.0.1',
                    '--port',
                    4222,
                    '--server-ca',
                    '/var/vcap/jobs/nats-tls/config/external_tls/ca.pem',
                    '--server-hostname',
                    'my-host',
                    '--client-certificate',
                    '/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem',
                    '--client-private-key',
                    '/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem',
                    '--user',
                    'my-user',
                    '--password',
                    'my-password'
                  ]
                }
              ]
            }

            expect(rendered_template).to eq(expected_template)
          end

          describe 'password authentication is disabled' do
            before do
              merged_manifest_properties['nats']['user'] = nil
              merged_manifest_properties['nats']['password'] = nil
            end

            it 'renders the template without password authentication properties' do
              rendered_template = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
              expected_template = {
                'processes' => [
                  {
                    'name' => 'nats-tls-wrapper',
                    'limits' => {
                      'open_files' => 100000
                    },
                    'executable' => '/var/vcap/packages/nats-v2-migrate/bin/nats-wrapper',
                    'args' => ['--config-file', '/var/vcap/jobs/nats-tls/config/migrator-config.json']
                  },
                  {
                    'name' => 'healthcheck',
                    'executable' => '/var/vcap/packages/nats-tls-healthcheck/bin/nats-tls-healthcheck',
                    'args' => [
                      '--address',
                      '10.0.0.1',
                      '--port',
                      4222,
                      '--server-ca',
                      '/var/vcap/jobs/nats-tls/config/external_tls/ca.pem',
                      '--server-hostname',
                      'my-host',
                      '--client-certificate',
                      '/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem',
                      '--client-private-key',
                      '/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem',
                    ]
                  }
                ]
              }

              expect(rendered_template).to eq(expected_template)
            end
          end
        end

        describe 'config/client_tls/certificate.pem' do
          let(:template) { job.template('config/client_tls/certificate.pem') }

          it 'renders the certificate correctly' do
            output = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
            expect(output).to eq('client-tls-cert')
          end

          describe 'when a client certificate is not provided in the manifest' do
            let(:merged_manifest_properties_without_certificate) do
              merged_manifest_properties.tap do |props|
                props['nats']['client']['tls'].delete('certificate')
              end
            end

            it 'fails with a meaningful error message' do
              expect do
                YAML.load(template.render(merged_manifest_properties_without_certificate, consumes: links, spec: spec))
              end.to raise_error(/nats.client.tls.certificate not provided in nats-tls job properties/)
            end
          end
        end
        describe 'config/client_tls/private_key.pem' do
          let(:template) { job.template('config/client_tls/private_key.pem') }

          it 'renders the private key correctly' do
            output = YAML.load(template.render(merged_manifest_properties, consumes: links, spec: spec))
            expect(output).to eq('client-tls-key')
          end

          describe 'when a client private_key is not provided in the manifest' do
            let(:merged_manifest_properties_without_private_key) do
              merged_manifest_properties.tap do |props|
                props['nats']['client']['tls'].delete('private_key')
              end
            end

            it 'fails with a meaningful error message' do
              expect do
                YAML.load(template.render(merged_manifest_properties_without_private_key, consumes: links, spec: spec))
              end.to raise_error(/nats.client.tls.private_key not provided in nats-tls job properties/)
            end
          end
        end
      end
    end
  end
end
