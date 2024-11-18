require 'bundler/setup'

lib_dir = File.join(File.expand_path(__dir__), '../pb')
$LOAD_PATH.unshift(lib_dir) unless $LOAD_PATH.include?(lib_dir)

require 'echo/echo_services_pb'

class EchoImpl < Echo::EchoService::Service
  def echo(req, call)
    puts req.inspect
    puts call.inspect
  end
end

class GrpcServer
  class << self
    def start
      @server = GRPC::RpcServer.new(
        pool_size: pool_size,
        max_waiting_requests: 1000,
        pool_keep_alive: 1800
      )

      @server.add_http2_port('0.0.0.0:50051', :this_port_is_insecure)
      @server.handle(EchoImpl.new)

      # Fork workers
      workers = []
      worker_count.times do
        workers << fork do
          setup_connection_pool
          @server.run
        end
      end

      # Signal handling
      %w[INT TERM].each do |sig|
        Signal.trap(sig) do
          workers.each { |pid| Process.kill(sig, pid) }
          exit
        end
      end

      workers.each { |pid| Process.wait(pid) }
    end

    private

    def worker_count
      ENV.fetch('WORKER_COUNT', 4).to_i
    end

    def pool_size
      ENV.fetch('POOL_SIZE', 30).to_i
    end

    def setup_connection_pool
      # Reset AR connection pool for forked process
      # ActiveRecord::Base.connection_pool.disconnect!

      # ActiveRecord::Base.connection_pool = ConnectionPool.new(size: pool_size) do
      #   ActiveRecord::Base.establish_connection(
      #     YAML.load_file('config/database.yml')[ENV['RACK_ENV'] || 'development']
      #   )
      #   ActiveRecord::Base.connection
      # end
    end
  end
end

GrpcServer.start
