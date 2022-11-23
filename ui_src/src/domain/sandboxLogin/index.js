// Credit for The NATS.IO Authors
// Copyright 2021-2022 The Memphis Authors
// Licensed under the Apache License, Version 2.0 (the “License”);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an “AS IS” BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package server

import './style.scss';

import React, { useState, useContext, useEffect } from 'react';
import { useHistory } from 'react-router-dom';
import { Form } from 'antd';

import { LOCAL_STORAGE_TOKEN } from '../../const/localStorageConsts';
import betaFullLogo from '../../assets/images/betaFullLogo.svg';
import { ApiEndpoints } from '../../const/apiEndpoints';
import sharps from '../../assets/images/sharps.svg';
import { httpRequest } from '../../services/http';
import AuthService from '../../services/auth';
import Button from '../../components/button';
import Loader from '../../components/loader';
import GitHubLogo from '../../assets/images/githubLogo.svg';
import GoogleLogo from '../../assets/images/GoogleLogo.png';
import { Context } from '../../hooks/store';
import Input from '../../components/Input';
import { GOOGLE_CLIENT_ID, GITHUB_CLIENT_ID, REDIRECT_URI, SOCKET_URL, URL } from '../../config';
import { connect } from 'nats.ws';

const SandboxLogin = (props) => {
    const [state, dispatch] = useContext(Context);
    const history = useHistory();
    const [loginForm] = Form.useForm(); // form controller
    const [isLoading, setisLoading] = useState(false);
    const [formFields, setFormFields] = useState({
        username: '',
        password: ''
    });
    const [error, setError] = useState('');
    const referer = props?.location?.state?.referer || '/overview';
    const [loadingSubmit, setLoadingSubmit] = useState(false);

    const signinWithGithub = async (code) => {
        try {
            setisLoading(true);
            const data = await httpRequest(
                'POST',
                ApiEndpoints.SANDBOX_LOGIN,
                {
                    login_type: 'github',
                    token: code
                },
                {},
                {},
                false
            );
            AuthService.saveToLocalStorage(data);
            localStorage.setItem('profile_pic', data.profile_pic); // profile_pic is available only in sandbox env
            history.push(referer);
            setisLoading(false);
        } catch (err) {
            setisLoading(false);
            setError(err);
        }
    };

    const handleGithubButtonClick = () => {
        window.location.href = `https://github.com/login/oauth/authorize?client_id=${GITHUB_CLIENT_ID}&scope=user&redirect_uri=${REDIRECT_URI}`;
    };

    useEffect(() => {
        let splittedUrl;
        if (localStorage.getItem(LOCAL_STORAGE_TOKEN) && AuthService.isValidToken()) {
            history.push(referer);
        }
        const shouldSigninWithGoogle = URL.includes('?code=') && URL.includes('&scope=email');
        const shouldSigninWithGithub = URL.includes('?code=');
        if (shouldSigninWithGoogle) {
            splittedUrl = URL.split('?code=');
            window.history.pushState({}, null, splittedUrl[0]);
            if (splittedUrl.length > 1) {
                const code = splittedUrl[1].split('&scope=email')[0];
                handleGoogleSignin(code);
            } else {
                setError('Authentication with GitHub failed');
            }
        } else if (shouldSigninWithGithub) {
            splittedUrl = URL.split('?code=');
            window.history.pushState({}, null, splittedUrl[0]);
            if (splittedUrl.length > 1) {
                signinWithGithub(`${splittedUrl[1]}`);
            } else {
                setError('Authentication with GitHub failed');
            }
        }
    }, []);

    const handleUserNameChange = (e) => {
        setFormFields({ ...formFields, username: e.target.value });
    };

    const handleGoogleButtonClick = () => window.location.replace(getGoogleAuthUri());

    const handlePasswordChange = (e) => {
        setFormFields({ ...formFields, password: e.target.value });
    };

    function getGoogleAuthUri() {
        const rootUrl = `https://accounts.google.com/o/oauth2/v2/auth`;
        let base = URL,
            state = '';
        let i = base.indexOf('#');
        if (i > -1) {
            state = base.substring(i);
            base = base.substring(0, i);
        }

        const options = {
            redirect_uri: REDIRECT_URI,
            client_id: GOOGLE_CLIENT_ID,
            access_type: 'offline',
            response_type: 'code',
            prompt: 'consent',
            scope: ['https://www.googleapis.com/auth/userinfo.profile', 'https://www.googleapis.com/auth/userinfo.email'].join(' '),
            state: state
        };

        const qs = new URLSearchParams(options);

        return `${rootUrl}?${qs.toString()}`;
    }

    const handleSubmit = async (e) => {
        e.preventDefault();
        const values = await loginForm.validateFields();
        if (values?.errorFields) {
            return;
        } else {
            try {
                setLoadingSubmit(true);
                const { username, password } = formFields;
                const data = await httpRequest('POST', ApiEndpoints.LOGIN, { username, password }, {}, {}, false);
                if (data) {
                    AuthService.saveToLocalStorage(data);
                    try {
                        const conn = await connect({
                            servers: [SOCKET_URL],
                            token: 'memphis'
                        });
                        dispatch({ type: 'SET_SOCKET_DETAILS', payload: conn });
                    } catch (error) {}
                    dispatch({ type: 'SET_USER_DATA', payload: data });
                    history.push(referer);
                }
            } catch (err) {
                setError(err);
            }
            setLoadingSubmit(false);
        }
    };

    const handleGoogleSignin = async (token) => {
        try {
            setisLoading(true);
            const data = await httpRequest(
                'POST',
                ApiEndpoints.SANDBOX_LOGIN,
                {
                    login_type: 'google',
                    token: token
                },
                {},
                {},
                false
            );
            AuthService.saveToLocalStorage(data);
            localStorage.setItem('profile_pic', data.profile_pic); // profile_pic is available only in sandbox env
            history.push(referer);
            setisLoading(false);
        } catch (err) {
            setisLoading(false);
            setError(err);
        }
    };

    const layout = {
        labelCol: {
            span: 8
        },
        wrapperCol: {
            span: 16
        }
    };

    const tailLayout = {
        wrapperCol: {
            offset: 8,
            span: 16
        }
    };

    return (
        <section className="sandbox-containers">
            {isLoading ? (
                <Loader></Loader>
            ) : (
                <div className="desktop-container">
                    <div className="desktop-content">
                        <div className="logoImg">
                            <img alt="logo" src={betaFullLogo}></img>
                        </div>
                        <content is="x3d">
                            <div className="title">
                                <p>Hey Memphiser,</p>
                                <p>Come try us</p>
                            </div>
                            <div className="login-container">
                                <div>
                                    <div className="sandbox-pad">
                                        <button type="button" className="google-login-button" onClick={handleGoogleButtonClick}>
                                            <img src={GoogleLogo} alt="git" className="git-image"></img>
                                            Sign in with Google
                                        </button>
                                        <button type="button" className="github-login-button" onClick={handleGithubButtonClick}>
                                            <img src={GitHubLogo} alt="git" className="git-image"></img>
                                            Sign in with GitHub
                                        </button>
                                    </div>
                                </div>
                                <or is="x3d">
                                    <span>or</span>
                                </or>
                                <div className="login-form">
                                    <Form
                                        {...layout}
                                        name="basic"
                                        initialValues={{
                                            remember: true
                                        }}
                                        form={loginForm}
                                    >
                                        <Form.Item
                                            name="username"
                                            rules={[
                                                {
                                                    required: true,
                                                    message: 'Username can not be empty'
                                                }
                                            ]}
                                        >
                                            <div className="field name">
                                                <p>Username</p>
                                                <Input
                                                    placeholder="Type username"
                                                    type="text"
                                                    radiusType="semi-round"
                                                    colorType="gray"
                                                    backgroundColorType="none"
                                                    borderColorType="gray"
                                                    width="19vw"
                                                    height="43px"
                                                    minWidth="200px"
                                                    onBlur={handleUserNameChange}
                                                    onChange={handleUserNameChange}
                                                    value={formFields.username}
                                                />
                                            </div>
                                        </Form.Item>
                                        <Form.Item
                                            name="password"
                                            rules={[
                                                {
                                                    required: true,
                                                    message: 'Password can not be empty'
                                                }
                                            ]}
                                        >
                                            <div className="field password">
                                                <p>Password</p>
                                                <Input
                                                    placeholder="Password"
                                                    type="password"
                                                    radiusType="semi-round"
                                                    colorType="gray"
                                                    backgroundColorType="none"
                                                    borderColorType="gray"
                                                    width="19vw"
                                                    height="43px"
                                                    minWidth="200px"
                                                    onChange={handlePasswordChange}
                                                    onBlur={handlePasswordChange}
                                                    value={formFields.password}
                                                />
                                            </div>
                                        </Form.Item>
                                        <Form.Item {...tailLayout} className="button-container">
                                            <Button
                                                width="19vw"
                                                height="43px"
                                                minWidth="200px"
                                                placeholder="Sign in"
                                                colorType="white"
                                                radiusType="circle"
                                                backgroundColorType="purple"
                                                fontSize="12px"
                                                fontWeight="600"
                                                isLoading={loadingSubmit}
                                                onClick={handleSubmit}
                                            />
                                        </Form.Item>

                                        {error && (
                                            <div className="error-message">
                                                <p>The username and password you entered did not match our records. Please double-check and try again.</p>
                                            </div>
                                        )}
                                    </Form>
                                </div>
                            </div>
                        </content>
                    </div>
                    <div className="brand-shapes">
                        <img alt="sharps" src={sharps}></img>
                    </div>
                </div>
            )}
        </section>
    );
};

export default SandboxLogin;
