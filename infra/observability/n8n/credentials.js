const required = (name) => {
  const value = process.env[name];
  if (!value) {
    throw new Error(`${name} is not set`);
  }

  return value;
};

const credentials = [
  {
    id: 'landPlotJira0001',
    name: 'Jira (land plot incidents)',
    type: 'jiraSoftwareCloudApi',
    data: {
      email: required('JIRA_EMAIL'),
      apiToken: required('JIRA_API_TOKEN'),
      domain: required('JIRA_BASE_URL').replace(/\/+$/, ''),
    },
  },
  {
    id: 'landPlotGemini01',
    name: 'Google Gemini (LLM_API_KEY)',
    type: 'googlePalmApi',
    data: {
      host: 'https://generativelanguage.googleapis.com',
      apiKey: process.env.LLM_API_KEY || 'not-set',
    },
  },
  {
    id: 'landPlotMailpit1',
    name: 'Mailpit SMTP',
    type: 'smtp',
    data: { user: '', password: '', host: 'mailpit', port: 1025, secure: false, disableStartTls: true },
  },
];

process.stdout.write(JSON.stringify(credentials));
