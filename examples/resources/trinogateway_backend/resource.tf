resource "trinogateway_backend" "example" {
  name          = "trino-1"
  proxy_to      = "http://localhost:8081"
  active        = true
  routing_group = "adhoc"
}
