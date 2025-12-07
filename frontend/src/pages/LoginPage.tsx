import { Auth } from '@supabase/auth-ui-react'
import { ThemeSupa } from '@supabase/auth-ui-shared'
import { supabase } from '../lib/supabase'

export default function LoginPage() {
    return (
        <div className="auth-container">
            <div className="card auth-card">
                <div style={{ textAlign: 'center', marginBottom: '2rem' }}>
                    <h1 className="logo" style={{ justifyContent: 'center', fontSize: '2rem' }}>DropLite</h1>
                    <p style={{ marginTop: '0.5rem' }}>安全、极简的文件传输服务</p>
                </div>
                <Auth
                    supabaseClient={supabase}
                    appearance={{
                        theme: ThemeSupa,
                        variables: {
                            default: {
                                colors: {
                                    brand: '#6366f1',
                                    brandAccent: '#4f46e5',
                                    inputBackground: 'rgba(15, 23, 42, 0.5)',
                                    inputText: 'white',
                                    inputBorder: 'rgba(148, 163, 184, 0.2)',
                                },
                            },
                        },
                        className: {
                            button: 'btn',
                            input: 'input',
                        }
                    }}
                    theme="dark"
                    providers={['github', 'google']}
                    localization={{
                        variables: {
                            sign_in: {
                                email_label: '邮箱地址',
                                password_label: '密码',
                                button_label: '登录',
                                loading_button_label: '登录中 ...',
                                email_input_placeholder: '您的邮箱',
                                password_input_placeholder: '您的密码',
                            },
                            sign_up: {
                                email_label: '邮箱地址',
                                password_label: '密码',
                                button_label: '注册',
                                loading_button_label: '注册中 ...',
                                email_input_placeholder: '您的邮箱',
                                password_input_placeholder: '您的密码',
                            }
                        }
                    }}
                />
            </div>
        </div>
    )
}
