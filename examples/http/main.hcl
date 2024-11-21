resource "container" "httpbin" {
  image {
    name = "kong/httpbin:0.1.0"
  }

  port {
    local = 80
    host = 80
  }

  health_check {
    timeout = "30s"

    http {
      address = "http://localhost/get"
      success_codes = [200]
    }
  }
}

resource "http" "get" {
  method = "GET"

  url = "http://${resource.container.httpbin.container_name}/get"

  headers = {
    Accept = "application/json"
  }
}

resource "http" "post" {
  method = "POST"

  url = "http://${resource.container.httpbin.container_name}/post"

  payload = jsonencode({
    foo = "bar"
  })

  headers = {
    Accept = "application/json"
  }
}

output "get_body" {
  value = jsondecode(resource.http.get.body)
}

output "get_status" {
  value = resource.http.get.status
}

output "post_body" {
  value = jsondecode(resource.http.post.body)
}

output "post_status" {
  value = resource.http.post.status
}