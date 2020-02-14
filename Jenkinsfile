pipeline {
    agent {
        dockerfile true
    }

    stages {
        stage('Lint') {
            steps {
                sh 'golangci-lint run'
            }
        }
        stage('Test') {
            steps {
                sh 'make test'
            }
        }
        stage('Build') {
            steps {
                sh 'make cross-build'
                archiveArtifacts 'dist/'
            }
        }
    }
}
