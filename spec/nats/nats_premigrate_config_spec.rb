require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'premigrate.conf.erb' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:merged_manifest_properties) {}

      let(:links) do
        [
          Link.new(
            name: 'nats-tls',
            instances: [
              LinkInstance.new(id: 'abc123'),
              LinkInstance.new(id: 'def456')
            ],
            properties: {
              'nats' => {
                'user' => 'my-user',
                'password' => 'my-password',
                'hostname' => 'nats.service.cf.internal',
                'port' => 4224,
                'cluster_port' => 4225,
                'http' => '0.0.0.0:0'
              }
            }
          )
        ]
      end

      let(:spec) do { 'address' => '10.0.0.1' } end

    describe "premigrate config for nats" do
     let(:job) {release.job('nats')}
     let(:template) { job.template('config/premigrate.conf') }

      it 'renders the template with the provided manifest properties' do
        rendered_template = template.render(merged_manifest_properties, consumes: links, spec: spec)
        rendered_struct = JSON.parse(rendered_template)

        expect(rendered_struct["nats_machines"]). to eq(["abc123.nats.service.cf.internal:4224", "def456.nats.service.cf.internal:4224"])
        expect(rendered_struct["nats_bpm_config_path"]). to eq("/var/vcap/jobs/nats-tls/config/bpm.yml")
        expect(rendered_struct["nats_v1_bpm_config_path"]). to eq("/var/vcap/jobs/nats-tls/config/bpm.v1.yml")
      end
    end
    end
  end
end
