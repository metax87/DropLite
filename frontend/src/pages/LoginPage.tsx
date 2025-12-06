import { Auth } from '@supabase/auth-ui-react'
import { ThemeSupa } from '@supabase/auth-ui-shared'
import { supabase } from '../lib/supabase'

export default function LoginPage() {
    return (
        <div style={{
            maxWidth: 400,
            margin: '40px auto',
            padding: '20px',
            border: '1px solid #eaeaea',
            borderRadius: '8px',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
        }}>
            <h1 style={{ textAlign: 'center', marginBottom: '20px' }}>DropLite 登录</h1>
            <Auth
                supabaseClient={supabase}
                appearance={{ theme: ThemeSupa }}
                theme="default"
                providers={['github', 'google']}
            />
        </div>
    )
}
