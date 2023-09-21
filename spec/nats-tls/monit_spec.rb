require 'rspec'
require 'bosh/template/test'
require 'yaml'
require 'json'

module Bosh::Template::Test
  describe 'monit' do
    describe 'template rendering' do
      let(:release_path) { File.join(File.dirname(__FILE__), '../..') }
      let(:release) { ReleaseDir.new(release_path) }
      let(:job) {release.job('nats-tls')}
      let(:job_path) { job.instance_variable_get(:@job_path) }
      let(:spec) { job.instance_variable_get(:@spec) }
      let(:template) { Template.new(spec, File.join(job_path, 'monit')) }

      let(:merged_manifest_properties) do
        {
          'nats' => {
            'mem_limit' => {
              'alert' => '500 MB',
              'restart' => '3000 MB'
            }
          }
        }
      end

      describe 'defaults' do
        it 'renders the template with the provided manifest properties' do
          rendered_template = template.render({})
          expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
          expect(rendered_template).to include("if totalmem > 3000 MB then restart")
        end
      end
      describe 'alert limits' do
        describe 'set B limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 B'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 100 B for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 3000 MB then restart")
          end
        end
        describe 'set KB limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 KB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 100 KB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 3000 MB then restart")
          end
        end
        describe 'set MB limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 MB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 100 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 3000 MB then restart")
          end
        end
        describe 'set GB limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 GB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 100 GB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 3000 MB then restart")
          end
        end
        describe 'set % limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 %'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 100 % for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 3000 MB then restart")
          end
        end
      end
      describe 'restart limits' do
        describe 'set B limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 B'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 100 B then restart")
          end
        end
        describe 'set KB limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 KB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 100 KB then restart")
          end
        end
        describe 'set MB limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 MB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 100 MB then restart")
          end
        end
        describe 'set GB limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 GB'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 100 GB then restart")
          end
        end
        describe 'set % limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 %'
          end
          it 'renders the template with the provided manifest properties' do
            rendered_template = template.render(merged_manifest_properties)
            expect(rendered_template).to include("if totalmem > 500 MB for 2 cycles then alert")
            expect(rendered_template).to include("if totalmem > 100 % then restart")
          end
        end
      end
      describe 'errors' do
        describe 'set bad limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = '100 potatoes'
          end
          it 'raises an error' do
            expect do
              template.render(merged_manifest_properties)
            end.to raise_error(/Bad 'nats.mem_limit.alert' setting: 100 potatoes. Format is: <number> B|KB|MB|GB|%/)
          end
        end
        describe 'set non-integer limit on alert' do
          before do
            merged_manifest_properties['nats']['mem_limit']['alert'] = 'mashed potatoes'
          end
          it 'raises an error' do
            expect do
              template.render(merged_manifest_properties)
            end.to raise_error(/Bad 'nats.mem_limit.alert' setting: mashed potatoes. Format is: <number> B|KB|MB|GB|%/)
          end
        end
        describe 'set bad limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = '100 potatoes'
          end
          it 'raises an error' do
            expect do
              template.render(merged_manifest_properties)
            end.to raise_error(/Bad 'nats.mem_limit.restart' setting: 100 potatoes. Format is: <number> B|KB|MB|GB|%/)
          end
        end
        describe 'set non-integer limit on restart' do
          before do
            merged_manifest_properties['nats']['mem_limit']['restart'] = 'mashed potatoes'
          end
          it 'raises an error' do
            expect do
              template.render(merged_manifest_properties)
            end.to raise_error(/Bad 'nats.mem_limit.restart' setting: mashed potatoes. Format is: <number> B|KB|MB|GB|%/)
          end
        end
      end
    end
  end
end
