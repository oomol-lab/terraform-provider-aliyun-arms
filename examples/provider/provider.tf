terraform {
  required_providers {
    arms = {
      source  = "oomol-lab/aliyun-arms"
      version = "~> 0.1"
    }
  }
}

provider "arms" {
  region  = "ap-southeast-1"
  profile = "default"
}
