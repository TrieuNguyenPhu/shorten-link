output "name" { value = aws_dynamodb_table.links.name }
output "arn" { value = aws_dynamodb_table.links.arn }
output "index_arn" { value = "${aws_dynamodb_table.links.arn}/index/GSI1" }
