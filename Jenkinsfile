pipeline {
  agent any

  parameters{
    string(name: 'COMMIT_TAG', defaultValue: '', description: 'Release version')
  }
  stages {
    stage('Compile') {
      parallel {
        stage('Compile') {
          steps {
            sh 'make build/awstagger'
          }
        }
        stage('Test') {
          steps {
            sh 'make test'
          }
        }
      }
    }
  }
}
