def brickWebImage
def brickValidationImage
def brickStorageImage
def brickCAImage

def builds = ['BrickStorage', 'SonarScan Go', 'End SonarQube Analysis']

stage('Initial') {
    node('Linux') {
        deleteDir()
        checkout scm
        stash name: 'workspace_scm', useDefaultExcludes: false
    }
}
pipeline {
    options {
        disableConcurrentBuilds()
        buildDiscarder(logRotator(numToKeepStr: '10'))
        gitLabConnection('git.zd.datev.de')
        gitlabBuilds(builds: builds)
    }
    parameters {
        string defaultValue: '1.17', name: 'GOLANG_VERSION', trim: true
    }
    agent none
    post {
        changed {
            script {
                mail to: "moritz.basel@datev.de",
                subject: "[JENKINS] ${currentBuild.fullDisplayName}: ${currentBuild.currentResult}",
                body: """
BUILD_LOG:    ${env.BUILD_URL}
GIT_COMMIT: ${env.GIT_COMMIT}
NODE_NAME:    ${env.NODE_NAME}
STATE:            ${currentBuild.currentResult}
BRANCH:         ${env.BRANCH_NAME}
"""
            }
        }
    }
    triggers {
        upstream(upstreamProjects: "git.zd.datev.de/p252/swbase/golang/${params.GOLANG_VERSION}", threshold: hudson.model.Result.SUCCESS)
    }

    stages {
        stage('Linux Operations') {
            agent { label 'Linux'}
            environment {
                PATH = "${tool('dotnet-core')}:$PATH"
            }
            stages {
                stage('Unstash') {
                    steps {
                        deleteDir()
                        unstash 'workspace_scm'
                    }
                }
                stage('whitesourceScan') {
                    when {
                        branch 'datev-master'
                    }
                    steps {
                        whitesourceScan(configOpts:['nuget.resolveDependencies=true','nuget.resolveCsProjFiles=true','go.resolveDependencies=true','go.collectDependenciesAtRuntime=false','go.dependencyManager=modules'])
                    }
                }
                stage('Begin C# SonarAnalysis') {
                    steps {
                        killZombies()
                        sh 'dotnet new tool-manifest --force'
                        sh 'dotnet tool install dotnet-sonarscanner --version 5.2.0'

                        withEnv(["PATH+NODEJS=${tool('nodejs')}/bin", "TMPDIR=${env.WORKSPACE}/sonarqube_tmp"]) 
                           {
                            withSonarQubeEnv('sonar.bk.datev.de') 
                            {
                                script {
                                    version = readFile(file: "version.txt", encoding: 'UTF-8').trim()
                                    sh 'mkdir -p $TMPDIR'
                                    sh "dotnet sonarscanner begin /key:'int.prodkrypt.trustcenter.Brick' /v:'${version}' /d:sonar.login=$SONAR_AUTH_TOKEN /d:sonar.cs.vstest.reportsPaths='testresults/*.trx' \
                                    /d:sonar.cs.opencover.reportsPaths='testresults/**/coverage.opencover.xml' /d:sonar.host.url='https://sonar.bk.datev.de' /d:sonar.branch.name='${env.BRANCH_NAME}'"
                                }
                            }
                        }
                    }
                }
                stage('Build') {
                    steps {
                        gitlabCommitStatus(STAGE_NAME) {    //muss Projekte einzeiln bauen wegen https://github.com/dotnet/sdk/issues/7238
                            sh 'dotnet publish -r ubuntu.18.04-x64 -c Release -o release/brickca Brick.CA/Brick.CA/Brick.CA.csproj'
                            sh 'dotnet publish -r ubuntu.18.04-x64 -c Release -o release/brickstorage Brick.Storage/Brick.Storage/Brick.Storage.csproj'
                            dir('release') {
                                stash 'csharp_artifacts'
                            }
                            archive "release/**/*"
                        }
                    }
                }
                stage('Test') {
                    steps {
                        gitlabCommitStatus(STAGE_NAME) {
                            sh 'dotnet test -c Release --logger trx --collect "XPlat Code Coverage" --results-directory testresults --settings coverlet.runsettings'
                        }
                    }
                }
                stage('End SonarQube Analysis') {
                    steps {
                        gitlabCommitStatus(STAGE_NAME) {
                            withEnv(["PATH+NODEJS=${tool('nodejs')}/bin"]) {
                                withSonarQubeEnv('sonar.bk.datev.de') 
                                {
                                    sh 'dotnet sonarscanner end /d:sonar.login=$SONAR_AUTH_TOKEN'
                                }
                            }
                            stash name: 'sonar_reports', includes: '.sonarqube/**'
                        }
                    }
                }
            }
        }
        stage('Standalone Go Analysis') {
            stages {
                stage('Get Coverage') {
                    agent { label 'Docker' }
                    steps {
                        deleteDir()
                        unstash 'workspace_scm'

                        sh "docker run --rm -e CGO_ENABLED=0 -v \"${env.WORKSPACE}:/app\" -w /app repo.prod.datev.de/docker-datev-repo/datev/p252/swbase/golang:${params.GOLANG_VERSION} go test ./... -coverprofile=coverage.out -mod=vendor"
                        stash name: 'with_go_coverage', useDefaultExcludes: false
                    } 
                }
                stage('SonarScan Go') {
                    agent { label 'Linux' }
                    steps {
                        gitlabCommitStatus(STAGE_NAME) {
                            deleteDir()
                            unstash 'with_go_coverage'
                            script {
                                sh "mv sonar.properties sonar-project.properties"
                                sonarScan()
                            }
                        }
                    }
                }
            }
        }
        stage('Docker Operations') {
            agent { label 'Docker' }
            environment {
                DockerRegistryScheme = 'https'
                DockerRegistryHost = 'repo.prod.datev.de'
                DockerRegistryPath = 'docker-datev-repo/datev/pki'
            }
            stages {
                stage('BrickWeb') {
                    steps {
                        deleteDir()
                        unstash 'workspace_scm'
                        script {
                            docker.withRegistry("${env.DockerRegistryScheme}://${env.DockerRegistryHost}", 'ArtifactoryUser') {
                                brickWebImage = docker.build(
                                    "${env.DockerRegistryHost}/${env.DockerRegistryPath}/brickweb:${env.GIT_COMMIT}",
                                    "-f brickweb/Dockerfile --pull --build-arg BUILDER_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/swbase/golang:${params.GOLANG_VERSION} --build-arg RUNNER_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/osbase/debian/buster:master --label BUILD_URL_02=${env.BUILD_URL} --label GIT_URL=${env.GIT_URL} --label GIT_COMMIT=${env.GIT_COMMIT} ."
                                )
                                brickWebImage.push()
                                sh "/usr/bin/docker rmi -f ${DockerRegistryHost}/${DockerRegistryPath}/brickweb:${GIT_COMMIT}"
                            }
                        }
                    }
                }
                stage('Brickvalidation') {
                    steps {
                        script {
                            docker.withRegistry("${env.DockerRegistryScheme}://${env.DockerRegistryHost}", 'ArtifactoryUser') {
                                brickValidationImage = docker.build(
                                    "${env.DockerRegistryHost}/${env.DockerRegistryPath}/brickvalidation:${env.GIT_COMMIT}",
                                    "-f brickvalidation/Dockerfile --pull --build-arg BUILDER_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/swbase/golang:${params.GOLANG_VERSION} --build-arg RUNNER_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/osbase/debian/buster:master --label BUILD_URL_02=${env.BUILD_URL} --label GIT_URL=${env.GIT_URL} --label GIT_COMMIT=${env.GIT_COMMIT} ."
                                )
                                brickValidationImage.push()
                                sh "/usr/bin/docker rmi -f ${DockerRegistryHost}/${DockerRegistryPath}/brickvalidation:${GIT_COMMIT}"
                            }
                        }
                    }
                }
                stage('Unstash C#-artifacts') {
                    steps {
                        dir('release') {
                            unstash 'csharp_artifacts'
                        }
                    }
                }
                stage('BrickCA') {
                    steps {
                        script {
                            dir('release/brickca') {
                                docker.withRegistry("${env.DockerRegistryScheme}://${env.DockerRegistryHost}", 'ArtifactoryUser') {
                                    brickCAImage = docker.build(
                                        "${env.DockerRegistryHost}/${env.DockerRegistryPath}/brickca:${env.GIT_COMMIT}",
                                        "--pull --build-arg PARENT_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/swbase/safenet-nethsm --label BUILD_URL=${env.BUILD_URL} --label GIT_COMMIT=${env.GIT_COMMIT} ."
                                    )
                                    brickCAImage.push()
                                    sh "/usr/bin/docker rmi -f ${DockerRegistryHost}/${DockerRegistryPath}/brickca:${GIT_COMMIT}"
                                }
                            }
                        }
                    }
                }
                stage('BrickStorage') {
                    steps {
                        script {
                            dir('release/brickstorage') {
                                docker.withRegistry("${env.DockerRegistryScheme}://${env.DockerRegistryHost}", 'ArtifactoryUser') {
                                    brickStorageImage = docker.build(
                                        "${env.DockerRegistryHost}/${env.DockerRegistryPath}/brickstorage:${env.GIT_COMMIT}",
                                        "--pull --build-arg PARENT_IMAGE=repo.prod.datev.de/docker-datev-repo/datev/p252/osbase/debian/buster:master --label BUILD_URL=${env.BUILD_URL} --label GIT_COMMIT=${env.GIT_COMMIT} ."
                                    )
                                    brickStorageImage.push()
                                    sh "/usr/bin/docker rmi -f ${DockerRegistryHost}/${DockerRegistryPath}/brickstorage:${GIT_COMMIT}"
                                }
                            }
                        }
                    }
                }
            }
        }
        stage('Start CnRZ Deploy') {
            agent none
            when {
                branch 'datev-master'
            }
            steps {
                script {
                    git_commit = "${env.GIT_COMMIT}";
                }
                build job: '../brick-k8s/master', parameters: [[$class: 'StringParameterValue', name: 'DOCKER_TAG', value: String.valueOf(git_commit)]], wait: false
            }
        }
    }
}
