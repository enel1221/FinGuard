import { Box, Button, Card, CardContent, Typography } from '@mui/material';
import { Login as LoginIcon } from '@mui/icons-material';

export default function LoginPage() {
  return (
    <Box
      display="flex" justifyContent="center" alignItems="center"
      minHeight="100vh"
      sx={{ bgcolor: 'background.default' }}
    >
      <Card sx={{ maxWidth: 400, width: '100%' }}>
        <CardContent sx={{ textAlign: 'center', py: 4, px: 3 }}>
          <Typography variant="h4" fontWeight={700} color="primary" mb={1}>
            FinGuard
          </Typography>
          <Typography variant="body2" color="text.secondary" mb={4}>
            Multi-cloud FinOps platform
          </Typography>
          <Button
            variant="contained"
            size="large"
            startIcon={<LoginIcon />}
            href="/login"
            fullWidth
          >
            Sign in with SSO
          </Button>
          <Typography variant="caption" display="block" mt={2} color="text.secondary">
            Powered by Dex OIDC
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}
