import { useRef, useState } from 'react';
import {
  Dialog, DialogTitle, DialogContent, DialogActions, Button, TextField,
  FormControl, InputLabel, Select, MenuItem, Box, Typography, Divider,
  Accordion, AccordionSummary, AccordionDetails, Stepper, Step, StepLabel,
} from '@mui/material';
import { ExpandMore, UploadFile } from '@mui/icons-material';
import {
  api,
  type KubernetesSourceConfig,
  type AWSSourceConfig,
  type AzureSourceConfig,
  type GCPSourceConfig,
} from '../../lib/api';

type SourceType = 'kubernetes' | 'aws_account' | 'azure_subscription' | 'gcp_project';

interface Props {
  projectId: string;
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}

const SOURCE_LABELS: Record<SourceType, string> = {
  kubernetes: 'Kubernetes Cluster',
  aws_account: 'AWS Account',
  azure_subscription: 'Azure Subscription',
  gcp_project: 'GCP Project',
};

export default function AddSourceDialog({ projectId, open, onClose, onCreated }: Props) {
  const [step, setStep] = useState(0);
  const [name, setName] = useState('');
  const [type, setType] = useState<SourceType>('kubernetes');

  const [k8s, setK8s] = useState<KubernetesSourceConfig>({ clusterName: '', opencostUrl: '' });
  const [aws, setAws] = useState<AWSSourceConfig>({ accountId: '', roleArn: '', region: '' });
  const [azure, setAzure] = useState<AzureSourceConfig>({ subscriptionId: '', tenantId: '', clientId: '' });
  const [gcp, setGcp] = useState<GCPSourceConfig>({ projectId: '' });

  const fileInputRef = useRef<HTMLInputElement>(null);

  const reset = () => {
    setStep(0);
    setName('');
    setType('kubernetes');
    setK8s({ clusterName: '', opencostUrl: '' });
    setAws({ accountId: '', roleArn: '', region: '' });
    setAzure({ subscriptionId: '', tenantId: '', clientId: '' });
    setGcp({ projectId: '' });
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const canProceed = () => {
    if (step === 0) return name.trim().length > 0;
    return isConfigValid();
  };

  const isConfigValid = (): boolean => {
    switch (type) {
      case 'kubernetes':
        return k8s.clusterName.trim().length > 0 && k8s.opencostUrl.trim().length > 0;
      case 'aws_account':
        return aws.accountId.trim().length > 0 && aws.roleArn.trim().length > 0 && aws.region.trim().length > 0;
      case 'azure_subscription':
        return azure.subscriptionId.trim().length > 0 && azure.tenantId.trim().length > 0 && azure.clientId.trim().length > 0;
      case 'gcp_project':
        return gcp.projectId.trim().length > 0;
    }
  };

  const getConfig = (): Record<string, unknown> => {
    switch (type) {
      case 'kubernetes': return stripEmpty(k8s as unknown as Record<string, unknown>);
      case 'aws_account': return stripEmpty(aws as unknown as Record<string, unknown>);
      case 'azure_subscription': return stripEmpty(azure as unknown as Record<string, unknown>);
      case 'gcp_project': return stripEmpty(gcp as unknown as Record<string, unknown>);
    }
  };

  const handleCreate = async () => {
    await api.post(`/projects/${projectId}/sources`, {
      name,
      type,
      config: getConfig(),
    });
    handleClose();
    onCreated();
  };

  const handleFileUpload = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      setK8s(prev => ({ ...prev, kubeconfigRef: reader.result as string }));
    };
    reader.readAsText(file);
    e.target.value = '';
  };

  return (
    <Dialog open={open} onClose={handleClose} maxWidth="sm" fullWidth>
      <DialogTitle>Add Cost Source</DialogTitle>
      <DialogContent>
        <Stepper activeStep={step} sx={{ mt: 1, mb: 3 }}>
          <Step><StepLabel>Basic Info</StepLabel></Step>
          <Step><StepLabel>Configuration</StepLabel></Step>
        </Stepper>

        {step === 0 && (
          <Box>
            <TextField
              fullWidth
              label="Source Name"
              value={name}
              onChange={e => setName(e.target.value)}
              sx={{ mb: 2 }}
            />
            <FormControl fullWidth>
              <InputLabel>Source Type</InputLabel>
              <Select
                value={type}
                label="Source Type"
                onChange={e => setType(e.target.value as SourceType)}
              >
                {(Object.entries(SOURCE_LABELS) as [SourceType, string][]).map(([val, label]) => (
                  <MenuItem key={val} value={val}>{label}</MenuItem>
                ))}
              </Select>
            </FormControl>
          </Box>
        )}

        {step === 1 && type === 'kubernetes' && (
          <KubernetesForm config={k8s} onChange={setK8s} fileInputRef={fileInputRef} onFileUpload={handleFileUpload} />
        )}
        {step === 1 && type === 'aws_account' && (
          <AWSForm config={aws} onChange={setAws} />
        )}
        {step === 1 && type === 'azure_subscription' && (
          <AzureForm config={azure} onChange={setAzure} />
        )}
        {step === 1 && type === 'gcp_project' && (
          <GCPForm config={gcp} onChange={setGcp} />
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={handleClose}>Cancel</Button>
        {step > 0 && <Button onClick={() => setStep(0)}>Back</Button>}
        {step === 0 ? (
          <Button variant="contained" onClick={() => setStep(1)} disabled={!canProceed()}>
            Next
          </Button>
        ) : (
          <Button variant="contained" onClick={handleCreate} disabled={!canProceed()}>
            Create Source
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
}

function KubernetesForm({ config, onChange, fileInputRef, onFileUpload }: {
  config: KubernetesSourceConfig;
  onChange: (c: KubernetesSourceConfig) => void;
  fileInputRef: React.RefObject<HTMLInputElement | null>;
  onFileUpload: (e: React.ChangeEvent<HTMLInputElement>) => void;
}) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" mb={2}>
        Connect to a Kubernetes cluster via OpenCost to collect workload cost data.
      </Typography>
      <TextField
        fullWidth required label="Cluster Name"
        value={config.clusterName}
        onChange={e => onChange({ ...config, clusterName: e.target.value })}
        helperText="A unique identifier for this cluster"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth required label="OpenCost URL"
        value={config.opencostUrl}
        onChange={e => onChange({ ...config, opencostUrl: e.target.value })}
        placeholder="http://opencost.opencost.svc.cluster.local:9003"
        helperText="The OpenCost API endpoint accessible from FinGuard"
        sx={{ mb: 2 }}
      />
      <Divider sx={{ my: 2 }} />
      <Typography variant="body2" fontWeight={600} mb={1}>Kubeconfig (optional)</Typography>
      <Typography variant="caption" color="text.secondary" display="block" mb={1}>
        Provide a kubeconfig to authenticate with the cluster. You can paste it directly or upload a file.
      </Typography>
      <Box sx={{ display: 'flex', gap: 1, mb: 1 }}>
        <Button
          size="small"
          variant="outlined"
          startIcon={<UploadFile />}
          onClick={() => fileInputRef.current?.click()}
        >
          Upload File
        </Button>
        <input
          ref={fileInputRef as React.RefObject<HTMLInputElement>}
          type="file"
          accept=".yaml,.yml,.conf,.kubeconfig"
          hidden
          onChange={onFileUpload}
        />
      </Box>
      <TextField
        fullWidth multiline minRows={4} maxRows={12}
        label="Kubeconfig YAML"
        value={config.kubeconfigRef || ''}
        onChange={e => onChange({ ...config, kubeconfigRef: e.target.value })}
        placeholder="apiVersion: v1\nkind: Config\n..."
        sx={{ fontFamily: 'monospace' }}
        slotProps={{ htmlInput: { style: { fontFamily: 'monospace', fontSize: '0.8rem' } } }}
      />
    </Box>
  );
}

function AWSForm({ config, onChange }: {
  config: AWSSourceConfig;
  onChange: (c: AWSSourceConfig) => void;
}) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" mb={2}>
        Connect to an AWS account to collect cost and usage data via Cost and Usage Reports.
      </Typography>
      <TextField
        fullWidth required label="Account ID"
        value={config.accountId}
        onChange={e => onChange({ ...config, accountId: e.target.value })}
        helperText="12-digit AWS account ID"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth required label="Role ARN"
        value={config.roleArn}
        onChange={e => onChange({ ...config, roleArn: e.target.value })}
        placeholder="arn:aws:iam::123456789012:role/FinGuardCostRole"
        helperText="IAM role ARN that FinGuard will assume to read billing data"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth required label="Region"
        value={config.region}
        onChange={e => onChange({ ...config, region: e.target.value })}
        placeholder="us-east-1"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth label="External ID"
        value={config.externalId || ''}
        onChange={e => onChange({ ...config, externalId: e.target.value })}
        helperText="Optional external ID for cross-account role assumption"
        sx={{ mb: 2 }}
      />

      <Accordion variant="outlined" disableGutters>
        <AccordionSummary expandIcon={<ExpandMore />}>
          <Typography variant="body2">Athena Configuration (optional)</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <TextField fullWidth label="Athena S3 Bucket" value={config.athenaBucket || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, athenaBucket: e.target.value })} />
          <TextField fullWidth label="Athena Region" value={config.athenaRegion || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, athenaRegion: e.target.value })} />
          <TextField fullWidth label="Athena Database" value={config.athenaDatabase || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, athenaDatabase: e.target.value })} />
          <TextField fullWidth label="Athena Table" value={config.athenaTable || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, athenaTable: e.target.value })} />
          <TextField fullWidth label="Athena Workgroup" value={config.athenaWorkgroup || ''} sx={{ mb: 1 }}
            onChange={e => onChange({ ...config, athenaWorkgroup: e.target.value })} />
        </AccordionDetails>
      </Accordion>

      <TextField fullWidth label="CUR Version" value={config.curVersion || ''} sx={{ mt: 2 }}
        onChange={e => onChange({ ...config, curVersion: e.target.value })}
        helperText="Optional CUR report version" />
    </Box>
  );
}

function AzureForm({ config, onChange }: {
  config: AzureSourceConfig;
  onChange: (c: AzureSourceConfig) => void;
}) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" mb={2}>
        Connect to an Azure subscription to collect cost data via the Cost Management API.
      </Typography>
      <TextField
        fullWidth required label="Subscription ID"
        value={config.subscriptionId}
        onChange={e => onChange({ ...config, subscriptionId: e.target.value })}
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth required label="Tenant ID"
        value={config.tenantId}
        onChange={e => onChange({ ...config, tenantId: e.target.value })}
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth required label="Client ID"
        value={config.clientId}
        onChange={e => onChange({ ...config, clientId: e.target.value })}
        helperText="Azure AD application (service principal) client ID"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth label="Client Secret" type="password"
        value={config.clientSecret || ''}
        onChange={e => onChange({ ...config, clientSecret: e.target.value })}
        sx={{ mb: 2 }}
      />

      <Accordion variant="outlined" disableGutters>
        <AccordionSummary expandIcon={<ExpandMore />}>
          <Typography variant="body2">Storage Export Configuration (optional)</Typography>
        </AccordionSummary>
        <AccordionDetails>
          <TextField fullWidth label="Storage Account" value={config.storageAccount || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, storageAccount: e.target.value })} />
          <TextField fullWidth label="Storage Access Key" type="password" value={config.storageAccessKey || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, storageAccessKey: e.target.value })} />
          <TextField fullWidth label="Storage Container" value={config.storageContainer || ''} sx={{ mb: 2 }}
            onChange={e => onChange({ ...config, storageContainer: e.target.value })} />
          <TextField fullWidth label="Container Path" value={config.containerPath || ''} sx={{ mb: 1 }}
            onChange={e => onChange({ ...config, containerPath: e.target.value })} />
        </AccordionDetails>
      </Accordion>

      <FormControl fullWidth sx={{ mt: 2 }}>
        <InputLabel>Azure Cloud</InputLabel>
        <Select
          value={config.azureCloud || ''}
          label="Azure Cloud"
          onChange={e => onChange({ ...config, azureCloud: e.target.value })}
        >
          <MenuItem value="">Default (Public)</MenuItem>
          <MenuItem value="AzurePublicCloud">Azure Public Cloud</MenuItem>
          <MenuItem value="AzureUSGovernmentCloud">Azure US Government</MenuItem>
          <MenuItem value="AzureChinaCloud">Azure China</MenuItem>
        </Select>
      </FormControl>
    </Box>
  );
}

function GCPForm({ config, onChange }: {
  config: GCPSourceConfig;
  onChange: (c: GCPSourceConfig) => void;
}) {
  return (
    <Box>
      <Typography variant="body2" color="text.secondary" mb={2}>
        Connect to a GCP project to collect billing data via BigQuery billing export.
      </Typography>
      <TextField
        fullWidth required label="GCP Project ID"
        value={config.projectId}
        onChange={e => onChange({ ...config, projectId: e.target.value })}
        helperText="The GCP project ID where billing export is configured"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth label="Billing Account ID"
        value={config.billingAccountId || ''}
        onChange={e => onChange({ ...config, billingAccountId: e.target.value })}
        placeholder="01ABCD-234EFG-567HIJ"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth label="Billing Data Dataset"
        value={config.billingDataDataset || ''}
        onChange={e => onChange({ ...config, billingDataDataset: e.target.value })}
        placeholder="project.dataset.table"
        helperText="BigQuery dataset containing billing export"
        sx={{ mb: 2 }}
      />
      <TextField
        fullWidth multiline minRows={3} maxRows={10}
        label="Service Account Key JSON"
        value={config.serviceAccountKey || ''}
        onChange={e => onChange({ ...config, serviceAccountKey: e.target.value })}
        placeholder='{"type": "service_account", ...}'
        helperText="Paste the JSON key for a service account with billing read access"
        slotProps={{ htmlInput: { style: { fontFamily: 'monospace', fontSize: '0.8rem' } } }}
      />
    </Box>
  );
}

function stripEmpty(obj: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(obj)) {
    if (v !== '' && v !== undefined && v !== null) {
      result[k] = v;
    }
  }
  return result;
}
