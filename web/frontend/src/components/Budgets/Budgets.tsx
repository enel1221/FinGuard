import { Box, Typography, Card, CardContent } from '@mui/material';

export default function Budgets() {
  return (
    <Box>
      <Typography variant="h5" fontWeight={600} mb={3}>Budgets</Typography>
      <Card>
        <CardContent>
          <Typography variant="body2" color="text.secondary">
            Budget tracking is configured per-project and per-cost-source. Create a project
            and add cost sources to begin monitoring spend against budgets.
          </Typography>
        </CardContent>
      </Card>
    </Box>
  );
}
